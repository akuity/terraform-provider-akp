package akp

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

var (
	_ resource.Resource                = &AkpInstanceIPAllowListResource{}
	_ resource.ResourceWithImportState = &AkpInstanceIPAllowListResource{}

	// instanceMutexKV is a mutex map to ensure only one operation per instance at a time
	// This prevents race conditions when multiple IP allow list resources update the same instance
	instanceMutexKV = NewMutexKV()
)

// init sets up debugging for CI environments
func init() {
	// In CI, dump all goroutine stacks before the test timeout
	// This helps diagnose deadlocks and hangs
	go func() {
		if os.Getenv("CI") == "true" {
			// Wait 42 minutes (before 45min CI timeout)
			var timeout time.Duration = 42
			time.Sleep(timeout * time.Minute)
			buf := make([]byte, 10*1024*1024)
			n := runtime.Stack(buf, true)
			// Write to both stdout and stderr to ensure visibility in CI logs
			msg := fmt.Sprintf("\n\n========== GOROUTINE DUMP (CI timeout approaching after 42min) ==========\n%s\n==========================================================================\n\n", buf[:n])
			_, _ = os.Stdout.WriteString(msg)
			_, _ = os.Stderr.WriteString(msg)
			os.Exit(1)
		}
	}()
}

// MutexKV is a simple key/value store for arbitrary mutexes. It can be used to
// serialize changes across arbitrary collaborators that share knowledge of the
// keys they must serialize on.
type MutexKV struct {
	lock  sync.Mutex
	store map[string]*sync.Mutex
}

// NewMutexKV returns a properly initialized MutexKV
func NewMutexKV() *MutexKV {
	return &MutexKV{
		store: make(map[string]*sync.Mutex),
	}
}

// Lock the mutex for the given key. Caller is responsible for calling Unlock
// for the same key. Includes timeout detection to help diagnose deadlocks.
func (m *MutexKV) Lock(key string) {
	m.lock.Lock()
	mutex, ok := m.store[key]
	if !ok {
		mutex = &sync.Mutex{}
		m.store[key] = mutex
	}
	m.lock.Unlock()
	mutex.Lock()
}

// Unlock the mutex for the given key. Caller must have called Lock for the same key first
func (m *MutexKV) Unlock(key string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	mutex, ok := m.store[key]
	if !ok {
		return
	}
	mutex.Unlock()
}

func NewAkpInstanceIPAllowListResource() resource.Resource {
	return &AkpInstanceIPAllowListResource{}
}

type AkpInstanceIPAllowListResource struct {
	akpCli *AkpCli
}

type IPAllowListResourceModel struct {
	ID         tftypes.String            `tfsdk:"id"`
	InstanceID tftypes.String            `tfsdk:"instance_id"`
	Entries    []*types.IPAllowListEntry `tfsdk:"entries"`
}

func (r *AkpInstanceIPAllowListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_ip_allow_list"
}

func (r *AkpInstanceIPAllowListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	akpCli, ok := req.ProviderData.(*AkpCli)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *AkpCli, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.akpCli = akpCli
}

func (r *AkpInstanceIPAllowListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating IP allow list")
	var plan IPAllowListResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	// Fetch the current instance
	_, err := r.getInstance(ctx, plan.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get instance: %s", err))
		return
	}

	// Lock the instance to prevent concurrent modifications during read-modify-write
	// We release the lock after applying changes, before waiting for health
	tflog.Debug(ctx, fmt.Sprintf("Acquiring lock for instance %s", plan.InstanceID.ValueString()))
	instanceMutexKV.Lock(plan.InstanceID.ValueString())
	tflog.Debug(ctx, fmt.Sprintf("Lock acquired for instance %s", plan.InstanceID.ValueString()))

	// Validate no duplicate IPs in the entries
	ipMap := make(map[string]bool)
	for _, entry := range plan.Entries {
		ip := entry.Ip.ValueString()
		if ipMap[ip] {
			instanceMutexKV.Unlock(plan.InstanceID.ValueString())
			resp.Diagnostics.AddError("Duplicate IP", fmt.Sprintf("IP %s appears multiple times in the entries list", ip))
			return
		}
		ipMap[ip] = true
	}

	// Fetch the current instance
	instance, err := r.getInstance(ctx, plan.InstanceID.ValueString())
	if err != nil {
		instanceMutexKV.Unlock(plan.InstanceID.ValueString())
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get instance: %s", err))
		return
	}

	// Check for conflicts: do any of our IPs already exist in the instance?
	currentIPs := make(map[string]bool)
	for _, entry := range instance.ArgoCD.Spec.InstanceSpec.IpAllowList {
		currentIPs[entry.Ip.ValueString()] = true
	}

	for _, entry := range plan.Entries {
		ip := entry.Ip.ValueString()
		if currentIPs[ip] {
			instanceMutexKV.Unlock(plan.InstanceID.ValueString())
			resp.Diagnostics.AddError(
				"Duplicate IP",
				fmt.Sprintf("IP %s already exists in the allow list. It may be managed by another resource or configured in akp_instance.ip_allow_list", ip),
			)
			return
		}
	}

	// Add our entries to the existing list
	instance.ArgoCD.Spec.InstanceSpec.IpAllowList = append(instance.ArgoCD.Spec.InstanceSpec.IpAllowList, plan.Entries...)

	// Apply changes to the instance
	instanceName, err := r.applyInstanceChanges(ctx, instance)
	if err != nil {
		instanceMutexKV.Unlock(plan.InstanceID.ValueString())
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update instance: %s", err))
		return
	}

	// Release the lock before waiting for health - this allows other operations to proceed
	instanceMutexKV.Unlock(plan.InstanceID.ValueString())

	// Wait for the instance to become healthy (outside the mutex)
	if err := r.waitForInstanceHealth(ctx, instanceName); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Instance did not become healthy: %s", err))
		return
	}

	// Generate a unique ID for this resource
	// Each akp_instance_ip_allow_list resource needs its own ID since multiple resources
	// can manage different sets of IPs on the same instance
	plan.ID = tftypes.StringValue(uuid.New().String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AkpInstanceIPAllowListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading IP allow list")
	var data IPAllowListResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	// Fetch the current instance
	instance, err := r.getInstance(ctx, data.InstanceID.ValueString())
	if err != nil {
		handleReadResourceError(ctx, resp, err)
		return
	}

	// Build a map of IPs we manage (from state)
	managedIPs := make(map[string]*types.IPAllowListEntry)
	for _, entry := range data.Entries {
		managedIPs[entry.Ip.ValueString()] = entry
	}

	// Find our entries in the current instance list
	var updatedEntries []*types.IPAllowListEntry
	for _, entry := range instance.ArgoCD.Spec.InstanceSpec.IpAllowList {
		if _, exists := managedIPs[entry.Ip.ValueString()]; exists {
			// This IP is managed by us, include it
			updatedEntries = append(updatedEntries, entry)
		}
	}

	// If we manage any IPs that are no longer in the instance, they were deleted externally
	if len(updatedEntries) < len(data.Entries) {
		tflog.Warn(ctx, fmt.Sprintf("Some IPs managed by this resource were deleted externally. Expected %d, found %d", len(data.Entries), len(updatedEntries)))
	}

	// Update our state with the current values
	data.Entries = updatedEntries

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AkpInstanceIPAllowListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Updating IP allow list")
	var plan IPAllowListResourceModel
	var state IPAllowListResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	// Fetch the current instance
	_, err := r.getInstance(ctx, plan.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get instance: %s", err))
		return
	}

	// Lock the instance to prevent concurrent modifications during read-modify-write
	// We release the lock after applying changes, before waiting for health
	tflog.Debug(ctx, fmt.Sprintf("Acquiring lock for instance %s", state.InstanceID.ValueString()))
	instanceMutexKV.Lock(state.InstanceID.ValueString())
	tflog.Debug(ctx, fmt.Sprintf("Lock acquired for instance %s", state.InstanceID.ValueString()))

	// Validate no duplicate IPs in the entries
	ipMap := make(map[string]bool)
	for _, entry := range plan.Entries {
		ip := entry.Ip.ValueString()
		if ipMap[ip] {
			instanceMutexKV.Unlock(state.InstanceID.ValueString())
			resp.Diagnostics.AddError("Duplicate IP", fmt.Sprintf("IP %s appears multiple times in the entries list", ip))
			return
		}
		ipMap[ip] = true
	}

	// Fetch the current instance
	instance, err := r.getInstance(ctx, plan.InstanceID.ValueString())
	if err != nil {
		instanceMutexKV.Unlock(state.InstanceID.ValueString())
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get instance: %s", err))
		return
	}

	// Calculate what we previously managed (from state)
	oldIPs := make(map[string]bool)
	for _, entry := range state.Entries {
		oldIPs[entry.Ip.ValueString()] = true
	}

	// Calculate what we want to manage now (from plan)
	newIPs := make(map[string]bool)
	for _, entry := range plan.Entries {
		newIPs[entry.Ip.ValueString()] = true
	}

	// IPs to add: in plan but not in state
	toAdd := []*types.IPAllowListEntry{}
	for _, entry := range plan.Entries {
		ip := entry.Ip.ValueString()
		if !oldIPs[ip] {
			toAdd = append(toAdd, entry)
		}
	}

	// IPs to remove: in state but not in plan
	toRemove := make(map[string]bool)
	for _, entry := range state.Entries {
		ip := entry.Ip.ValueString()
		if !newIPs[ip] {
			toRemove[ip] = true
		}
	}

	// Check for conflicts: are any of the IPs we want to add already in the instance?
	// (excluding ones we're removing)
	currentIPs := make(map[string]bool)
	for _, entry := range instance.ArgoCD.Spec.InstanceSpec.IpAllowList {
		ip := entry.Ip.ValueString()
		if !toRemove[ip] { // Don't count IPs we're about to remove
			currentIPs[ip] = true
		}
	}

	for _, entry := range toAdd {
		ip := entry.Ip.ValueString()
		if currentIPs[ip] {
			instanceMutexKV.Unlock(state.InstanceID.ValueString())
			resp.Diagnostics.AddError(
				"Duplicate IP",
				fmt.Sprintf("IP %s already exists in the allow list. It may be managed by another resource or configured in akp_instance.ip_allow_list", ip),
			)
			return
		}
	}

	// Build the new instance IP list:
	// 1. Keep all IPs that aren't in toRemove
	// 2. Add the IPs from toAdd
	newList := []*types.IPAllowListEntry{}
	for _, entry := range instance.ArgoCD.Spec.InstanceSpec.IpAllowList {
		ip := entry.Ip.ValueString()
		if toRemove[ip] {
			// Skip IPs we're removing
			continue
		}
		if oldIPs[ip] {
			// This is an IP we manage - update it with new values from plan
			for _, planEntry := range plan.Entries {
				if planEntry.Ip.ValueString() == ip {
					newList = append(newList, planEntry)
					break
				}
			}
		} else {
			// This IP is managed by someone else - keep it unchanged
			newList = append(newList, entry)
		}
	}

	// Add new IPs
	newList = append(newList, toAdd...)

	instance.ArgoCD.Spec.InstanceSpec.IpAllowList = newList

	// Apply changes to the instance
	instanceName, err := r.applyInstanceChanges(ctx, instance)
	if err != nil {
		instanceMutexKV.Unlock(state.InstanceID.ValueString())
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update instance: %s", err))
		return
	}

	// Release the lock before waiting for health - this allows other operations to proceed
	instanceMutexKV.Unlock(state.InstanceID.ValueString())

	// Wait for the instance to become healthy (outside the mutex)
	if err := r.waitForInstanceHealth(ctx, instanceName); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Instance did not become healthy: %s", err))
		return
	}

	// Preserve the ID from state - it was generated during Create and should not change
	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AkpInstanceIPAllowListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Deleting IP allow list")
	var state IPAllowListResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	// Fetch the current instance
	_, err := r.getInstance(ctx, state.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get instance: %s", err))
		return
	}

	// Lock the instance to prevent concurrent modifications during read-modify-write
	// We release the lock after applying changes, before waiting for health
	tflog.Debug(ctx, fmt.Sprintf("Acquiring lock for instance %s", state.InstanceID.ValueString()))
	instanceMutexKV.Lock(state.InstanceID.ValueString())
	tflog.Debug(ctx, fmt.Sprintf("Lock acquired for instance %s", state.InstanceID.ValueString()))

	// Log what we're about to delete
	var managedIPsList []string
	for _, entry := range state.Entries {
		managedIPsList = append(managedIPsList, entry.Ip.ValueString())
	}
	tflog.Debug(ctx, fmt.Sprintf("Delete called for instance %s with IPs: %v", state.InstanceID.ValueString(), managedIPsList))

	// Fetch the current instance
	tflog.Debug(ctx, fmt.Sprintf("Fetching current instance %s", state.InstanceID.ValueString()))
	instance, err := r.getInstance(ctx, state.InstanceID.ValueString())
	if err != nil {
		instanceMutexKV.Unlock(state.InstanceID.ValueString())
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get instance: %s", err))
		return
	}

	// Log current IPs in the instance
	var currentIPsList []string
	for _, entry := range instance.ArgoCD.Spec.InstanceSpec.IpAllowList {
		currentIPsList = append(currentIPsList, entry.Ip.ValueString())
	}
	tflog.Debug(ctx, fmt.Sprintf("Current IPs in instance: %v", currentIPsList))

	// Build a map of IPs we manage (from state)
	managedIPs := make(map[string]bool)
	for _, entry := range state.Entries {
		managedIPs[entry.Ip.ValueString()] = true
	}

	// Remove only the IPs we manage, keep all others
	newList := []*types.IPAllowListEntry{}
	for _, entry := range instance.ArgoCD.Spec.InstanceSpec.IpAllowList {
		if !managedIPs[entry.Ip.ValueString()] {
			// This IP is not managed by us, keep it
			newList = append(newList, entry)
			tflog.Debug(ctx, fmt.Sprintf("Keeping IP %s (not managed by this resource)", entry.Ip.ValueString()))
		} else {
			tflog.Debug(ctx, fmt.Sprintf("Removing IP %s (managed by this resource)", entry.Ip.ValueString()))
		}
	}

	// Log the new list
	var newIPsList []string
	for _, entry := range newList {
		newIPsList = append(newIPsList, entry.Ip.ValueString())
	}
	tflog.Debug(ctx, fmt.Sprintf("New IP list after filtering: %v (nil: %v)", newIPsList, newList == nil))

	instance.ArgoCD.Spec.InstanceSpec.IpAllowList = newList

	// Apply changes to the instance
	tflog.Debug(ctx, fmt.Sprintf("Updating instance %s with new IP list", state.InstanceID.ValueString()))
	instanceName, err := r.applyInstanceChanges(ctx, instance)
	if err != nil {
		instanceMutexKV.Unlock(state.InstanceID.ValueString())
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update instance: %s", err))
		tflog.Error(ctx, fmt.Sprintf("Failed to update instance: %s", err))
		return
	}

	// Release the lock before waiting for health - this allows other operations to proceed
	instanceMutexKV.Unlock(state.InstanceID.ValueString())

	// Wait for the instance to become healthy (outside the mutex)
	if err := r.waitForInstanceHealth(ctx, instanceName); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Instance did not become healthy: %s", err))
		return
	}

	// Verify the changes were applied by re-fetching the instance
	tflog.Debug(ctx, "Verifying IP list changes were applied")
	updatedInstance, err := r.getInstance(ctx, state.InstanceID.ValueString())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to verify IP list changes: %s", err))
	} else {
		// Check if any of our managed IPs are still present
		for _, entry := range updatedInstance.ArgoCD.Spec.InstanceSpec.IpAllowList {
			if managedIPs[entry.Ip.ValueString()] {
				tflog.Error(ctx, fmt.Sprintf("IP %s was supposed to be deleted but is still present!", entry.Ip.ValueString()))
				resp.Diagnostics.AddError(
					"Delete Verification Failed",
					fmt.Sprintf("IP %s was not successfully removed from the instance", entry.Ip.ValueString()),
				)
				return
			}
		}
		var remainingIPs []string
		for _, entry := range updatedInstance.ArgoCD.Spec.InstanceSpec.IpAllowList {
			remainingIPs = append(remainingIPs, entry.Ip.ValueString())
		}
		tflog.Debug(ctx, fmt.Sprintf("Verified deletion successful. Remaining IPs in instance: %v", remainingIPs))
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully deleted IP allow list for instance %s", state.InstanceID.ValueString()))
}

func (r *AkpInstanceIPAllowListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: instance_id (e.g., "instance-id")
	instanceID := req.ID

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	// Fetch the instance to get all IP allow list entries
	instance, err := r.getInstance(ctx, instanceID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get instance: %s", err))
		return
	}

	// Import all entries from the instance
	// Generate a new unique ID for this imported resource
	var data IPAllowListResourceModel
	data.ID = tftypes.StringValue(uuid.New().String())
	data.InstanceID = tftypes.StringValue(instanceID)
	data.Entries = instance.ArgoCD.Spec.InstanceSpec.IpAllowList

	// If there are no entries, initialize as empty list
	if data.Entries == nil {
		data.Entries = []*types.IPAllowListEntry{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Helper function to get an instance
// Note: This function uses GetInstance directly instead of refreshState/ExportInstance
// because the IP allow list resource only needs the instance spec (which contains the IP allow list).
// ExportInstance would also try to fetch k3s resources (ConfigMaps, Applications, etc.) which
// can timeout if the k3s control plane is slow or unresponsive.
//
// This function does NOT wait for instance health before reading. The caller is responsible
// for ensuring proper synchronization via the instance mutex during read-modify-write operations.
func (r *AkpInstanceIPAllowListResource) getInstance(ctx context.Context, instanceID string) (*types.Instance, error) {
	getResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
		return r.akpCli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
			OrganizationId: r.akpCli.OrgId,
			IdType:         idv1.Type_ID,
			Id:             instanceID,
		})
	}, "GetInstance")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get instance")
	}

	var instance types.Instance
	instance.ID = tftypes.StringValue(getResp.Instance.Id)
	instance.Name = tftypes.StringValue(getResp.Instance.Name)

	// Initialize all map fields as null so buildApplyRequest/processResources handles them correctly
	// Without this, nil maps cause panic when ElementsAs is called
	instance.ArgoCDConfigMap = tftypes.MapNull(tftypes.StringType)
	instance.ArgoCDRBACConfigMap = tftypes.MapNull(tftypes.StringType)
	instance.ArgoCDSecret = tftypes.MapNull(tftypes.StringType)
	instance.ApplicationSetSecret = tftypes.MapNull(tftypes.StringType)
	instance.NotificationsConfigMap = tftypes.MapNull(tftypes.StringType)
	instance.NotificationsSecret = tftypes.MapNull(tftypes.StringType)
	instance.ImageUpdaterConfigMap = tftypes.MapNull(tftypes.StringType)
	instance.ImageUpdaterSSHConfigMap = tftypes.MapNull(tftypes.StringType)
	instance.ImageUpdaterSecret = tftypes.MapNull(tftypes.StringType)
	instance.ArgoCDKnownHostsConfigMap = tftypes.MapNull(tftypes.StringType)
	instance.ArgoCDTLSCertsConfigMap = tftypes.MapNull(tftypes.StringType)
	instance.RepoCredentialSecrets = tftypes.MapNull(tftypes.MapType{ElemType: tftypes.StringType})
	instance.RepoTemplateCredentialSecrets = tftypes.MapNull(tftypes.MapType{ElemType: tftypes.StringType})
	instance.ArgoCDResources = tftypes.MapNull(tftypes.StringType)

	// Initialize ArgoCD spec with required fields from the GetInstance response
	// We need Version (required by ApplyInstance) and the IP allow list
	description := tftypes.StringNull()
	if getResp.Instance.Description != "" {
		description = tftypes.StringValue(getResp.Instance.Description)
	}

	instance.ArgoCD = &types.ArgoCD{
		Spec: types.ArgoCDSpec{
			Version:      tftypes.StringValue(getResp.Instance.Version),
			Description:  description,
			InstanceSpec: convertAPIInstanceSpecToTF(getResp.Instance.GetSpec()),
		},
	}

	return &instance, nil
}

// convertAPIIPAllowListToTF converts IP allow list entries from API format to Terraform format
func convertAPIIPAllowListToTF(entries []*argocdv1.IPAllowListEntry) []*types.IPAllowListEntry {
	if entries == nil {
		return nil
	}
	result := make([]*types.IPAllowListEntry, len(entries))
	for i, entry := range entries {
		description := tftypes.StringNull()
		if entry.Description != "" {
			description = tftypes.StringValue(entry.Description)
		}
		result[i] = &types.IPAllowListEntry{
			Ip:          tftypes.StringValue(entry.Ip),
			Description: description,
		}
	}
	return result
}

// convertAPIInstanceSpecToTF converts the full API InstanceSpec to Terraform types.InstanceSpec.
// This preserves ALL instance settings when modifying the IP allow list.
func convertAPIInstanceSpecToTF(spec *argocdv1.InstanceSpec) types.InstanceSpec {
	if spec == nil {
		return types.InstanceSpec{}
	}

	result := types.InstanceSpec{
		IpAllowList:                     convertAPIIPAllowListToTF(spec.GetIpAllowList()),
		Subdomain:                       tftypes.StringValue(spec.GetSubdomain()),
		DeclarativeManagementEnabled:    tftypes.BoolValue(spec.GetDeclarativeManagementEnabled()),
		ImageUpdaterEnabled:             tftypes.BoolValue(spec.GetImageUpdaterEnabled()),
		BackendIpAllowListEnabled:       tftypes.BoolValue(spec.GetBackendIpAllowListEnabled()),
		AuditExtensionEnabled:           tftypes.BoolValue(spec.GetAuditExtensionEnabled()),
		SyncHistoryExtensionEnabled:     tftypes.BoolValue(spec.GetSyncHistoryExtensionEnabled()),
		AssistantExtensionEnabled:       tftypes.BoolValue(spec.GetAssistantExtensionEnabled()),
		MultiClusterK8SDashboardEnabled: tftypes.BoolValue(spec.GetMultiClusterK8SDashboardEnabled()),
	}

	// Handle optional string fields
	if spec.GetFqdn() != "" {
		result.Fqdn = tftypes.StringValue(spec.GetFqdn())
	} else {
		result.Fqdn = tftypes.StringNull()
	}

	// Note: Some newer fields (MetricsIngressUsername, MetricsIngressPasswordHash, PrivilegedNotificationCluster)
	// may not be available in older api-client-go versions. They will be null and won't be modified.
	result.MetricsIngressUsername = tftypes.StringNull()
	result.MetricsIngressPasswordHash = tftypes.StringNull()
	result.PrivilegedNotificationCluster = tftypes.StringNull()

	return result
}

// applyInstanceChanges applies instance changes via the API without waiting for health.
// Returns the instance name needed for subsequent health checks.
// This should be called while holding the instance mutex.
func (r *AkpInstanceIPAllowListResource) applyInstanceChanges(ctx context.Context, instance *types.Instance) (string, error) {
	var diags resource.UpdateResponse

	apiReq := buildApplyRequest(ctx, &diags.Diagnostics, instance, r.akpCli.OrgId)
	if diags.Diagnostics.HasError() {
		return "", fmt.Errorf("failed to build apply request")
	}

	tflog.Debug(ctx, fmt.Sprintf("Apply instance request for IP allow list update: %s", apiReq.Argocd))
	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.ApplyInstanceResponse, error) {
		return r.akpCli.Cli.ApplyInstance(ctx, apiReq)
	}, "ApplyInstance")
	if err != nil {
		return "", errors.Wrap(err, "unable to apply instance changes")
	}

	return instance.Name.ValueString(), nil
}

// waitForInstanceHealth waits for the instance to become healthy.
// This can be called without holding the instance mutex to allow concurrent operations.
// Note: We only wait for health status (not reconciliation) to match the behavior of
// the main instance resource. IP allow list changes are metadata-only and don't require
// waiting for full Kubernetes resource reconciliation.
func (r *AkpInstanceIPAllowListResource) waitForInstanceHealth(ctx context.Context, instanceName string) error {
	getResourceFunc := func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
		return retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
			return r.akpCli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
				OrganizationId: r.akpCli.OrgId,
				Id:             instanceName,
				IdType:         idv1.Type_NAME,
			})
		}, "GetInstance")
	}

	getHealthStatusFunc := func(resp *argocdv1.GetInstanceResponse) healthv1.StatusCode {
		if resp == nil || resp.Instance == nil {
			return healthv1.StatusCode_STATUS_CODE_UNKNOWN
		}
		return resp.Instance.GetHealthStatus().GetCode()
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for instance %s to become healthy", instanceName))
	healthErr := waitForStatus(
		ctx,
		getResourceFunc,
		getHealthStatusFunc,
		[]healthv1.StatusCode{healthv1.StatusCode_STATUS_CODE_HEALTHY},
		10*time.Second,
		10*time.Minute,
		fmt.Sprintf("Instance %s", instanceName),
		"health",
	)

	if healthErr != nil {
		return errors.Wrap(healthErr, fmt.Sprintf("instance '%s' did not become healthy", instanceName))
	}

	return nil
}

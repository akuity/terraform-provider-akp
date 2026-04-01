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
	"google.golang.org/protobuf/types/known/structpb"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

var (
	_ resource.Resource                = &AkpInstanceIPAllowListResource{}
	_ resource.ResourceWithImportState = &AkpInstanceIPAllowListResource{}

	instanceMutexKV = NewMutexKV()
)

func init() {
	go func() {
		if os.Getenv("CI") == "true" {
			var timeout time.Duration = 42
			time.Sleep(timeout * time.Minute)
			buf := make([]byte, 10*1024*1024)
			n := runtime.Stack(buf, true)
			msg := fmt.Sprintf("\n\n========== GOROUTINE DUMP (CI timeout approaching after 42min) ==========\n%s\n==========================================================================\n\n", buf[:n])
			_, _ = os.Stdout.WriteString(msg)
			_, _ = os.Stderr.WriteString(msg)
			os.Exit(1)
		}
	}()
}

type MutexKV struct {
	lock  sync.Mutex
	store map[string]*sync.Mutex
}

func NewMutexKV() *MutexKV {
	return &MutexKV{
		store: make(map[string]*sync.Mutex),
	}
}

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
	BaseResource
}

type IPAllowListResourceModel struct {
	ID         tftypes.String            `tfsdk:"id"`
	InstanceID tftypes.String            `tfsdk:"instance_id"`
	Entries    []*types.IPAllowListEntry `tfsdk:"entries"`
}

func (r *AkpInstanceIPAllowListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_ip_allow_list"
}

func (r *AkpInstanceIPAllowListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating IP allow list")
	var plan IPAllowListResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = r.AuthCtx(ctx)

	if _, _, err := getInstanceIPAllowList(ctx, r.akpCli, plan.InstanceID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get instance: %s", err))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Acquiring lock for instance %s", plan.InstanceID.ValueString()))
	instanceMutexKV.Lock(plan.InstanceID.ValueString())
	tflog.Debug(ctx, fmt.Sprintf("Lock acquired for instance %s", plan.InstanceID.ValueString()))

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

	currentEntries, instanceName, err := getInstanceIPAllowList(ctx, r.akpCli, plan.InstanceID.ValueString())
	if err != nil {
		instanceMutexKV.Unlock(plan.InstanceID.ValueString())
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get instance: %s", err))
		return
	}

	currentIPs := make(map[string]bool)
	for _, entry := range currentEntries {
		currentIPs[entry.Ip.ValueString()] = true
	}

	for _, entry := range plan.Entries {
		ip := entry.Ip.ValueString()
		if currentIPs[ip] {
			instanceMutexKV.Unlock(plan.InstanceID.ValueString())
			resp.Diagnostics.AddError(
				"Duplicate IP",
				fmt.Sprintf("IP %s already exists in the allow list. It may be managed by another resource", ip),
			)
			return
		}
	}

	newList := append(currentEntries, plan.Entries...)

	if err := patchInstanceIPAllowList(ctx, r.akpCli, plan.InstanceID.ValueString(), newList); err != nil {
		instanceMutexKV.Unlock(plan.InstanceID.ValueString())
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update instance: %s", err))
		return
	}

	instanceMutexKV.Unlock(plan.InstanceID.ValueString())

	if err := waitForInstanceHealth(ctx, r.akpCli, instanceName); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Instance did not become healthy: %s", err))
		return
	}

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

	ctx = r.AuthCtx(ctx)

	currentEntries, _, err := getInstanceIPAllowList(ctx, r.akpCli, data.InstanceID.ValueString())
	if err != nil {
		handleReadResourceError(ctx, resp, err)
		return
	}

	managedIPs := make(map[string]*types.IPAllowListEntry)
	for _, entry := range data.Entries {
		managedIPs[entry.Ip.ValueString()] = entry
	}

	var updatedEntries []*types.IPAllowListEntry
	for _, entry := range currentEntries {
		if _, exists := managedIPs[entry.Ip.ValueString()]; exists {
			updatedEntries = append(updatedEntries, entry)
		}
	}

	if len(updatedEntries) < len(data.Entries) {
		tflog.Warn(ctx, fmt.Sprintf("Some IPs managed by this resource were deleted externally. Expected %d, found %d", len(data.Entries), len(updatedEntries)))
	}

	if updatedEntries == nil {
		updatedEntries = []*types.IPAllowListEntry{}
	}
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

	ctx = r.AuthCtx(ctx)

	if _, _, err := getInstanceIPAllowList(ctx, r.akpCli, plan.InstanceID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get instance: %s", err))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Acquiring lock for instance %s", state.InstanceID.ValueString()))
	instanceMutexKV.Lock(state.InstanceID.ValueString())
	tflog.Debug(ctx, fmt.Sprintf("Lock acquired for instance %s", state.InstanceID.ValueString()))

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

	currentEntries, instanceName, err := getInstanceIPAllowList(ctx, r.akpCli, plan.InstanceID.ValueString())
	if err != nil {
		instanceMutexKV.Unlock(state.InstanceID.ValueString())
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get instance: %s", err))
		return
	}

	oldIPs := make(map[string]bool)
	for _, entry := range state.Entries {
		oldIPs[entry.Ip.ValueString()] = true
	}

	newIPs := make(map[string]bool)
	for _, entry := range plan.Entries {
		newIPs[entry.Ip.ValueString()] = true
	}

	toAdd := []*types.IPAllowListEntry{}
	for _, entry := range plan.Entries {
		ip := entry.Ip.ValueString()
		if !oldIPs[ip] {
			toAdd = append(toAdd, entry)
		}
	}

	toRemove := make(map[string]bool)
	for _, entry := range state.Entries {
		ip := entry.Ip.ValueString()
		if !newIPs[ip] {
			toRemove[ip] = true
		}
	}

	currentIPMap := make(map[string]bool)
	for _, entry := range currentEntries {
		ip := entry.Ip.ValueString()
		if !toRemove[ip] {
			currentIPMap[ip] = true
		}
	}

	for _, entry := range toAdd {
		ip := entry.Ip.ValueString()
		if currentIPMap[ip] {
			instanceMutexKV.Unlock(state.InstanceID.ValueString())
			resp.Diagnostics.AddError(
				"Duplicate IP",
				fmt.Sprintf("IP %s already exists in the allow list. It may be managed by another resource", ip),
			)
			return
		}
	}

	newList := []*types.IPAllowListEntry{}
	for _, entry := range currentEntries {
		ip := entry.Ip.ValueString()
		if toRemove[ip] {
			continue
		}
		if oldIPs[ip] {
			for _, planEntry := range plan.Entries {
				if planEntry.Ip.ValueString() == ip {
					newList = append(newList, planEntry)
					break
				}
			}
		} else {
			newList = append(newList, entry)
		}
	}

	newList = append(newList, toAdd...)

	if err := patchInstanceIPAllowList(ctx, r.akpCli, plan.InstanceID.ValueString(), newList); err != nil {
		instanceMutexKV.Unlock(state.InstanceID.ValueString())
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update instance: %s", err))
		return
	}

	instanceMutexKV.Unlock(state.InstanceID.ValueString())

	if err := waitForInstanceHealth(ctx, r.akpCli, instanceName); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Instance did not become healthy: %s", err))
		return
	}

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

	ctx = r.AuthCtx(ctx)

	if _, _, err := getInstanceIPAllowList(ctx, r.akpCli, state.InstanceID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get instance: %s", err))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Acquiring lock for instance %s", state.InstanceID.ValueString()))
	instanceMutexKV.Lock(state.InstanceID.ValueString())
	tflog.Debug(ctx, fmt.Sprintf("Lock acquired for instance %s", state.InstanceID.ValueString()))

	var managedIPsList []string
	for _, entry := range state.Entries {
		managedIPsList = append(managedIPsList, entry.Ip.ValueString())
	}
	tflog.Debug(ctx, fmt.Sprintf("Delete called for instance %s with IPs: %v", state.InstanceID.ValueString(), managedIPsList))

	tflog.Debug(ctx, fmt.Sprintf("Fetching current instance %s", state.InstanceID.ValueString()))
	currentEntries, instanceName, err := getInstanceIPAllowList(ctx, r.akpCli, state.InstanceID.ValueString())
	if err != nil {
		instanceMutexKV.Unlock(state.InstanceID.ValueString())
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get instance: %s", err))
		return
	}

	var currentIPsList []string
	for _, entry := range currentEntries {
		currentIPsList = append(currentIPsList, entry.Ip.ValueString())
	}
	tflog.Debug(ctx, fmt.Sprintf("Current IPs in instance: %v", currentIPsList))

	managedIPs := make(map[string]bool)
	for _, entry := range state.Entries {
		managedIPs[entry.Ip.ValueString()] = true
	}

	newList := []*types.IPAllowListEntry{}
	for _, entry := range currentEntries {
		if !managedIPs[entry.Ip.ValueString()] {
			newList = append(newList, entry)
			tflog.Debug(ctx, fmt.Sprintf("Keeping IP %s (not managed by this resource)", entry.Ip.ValueString()))
		} else {
			tflog.Debug(ctx, fmt.Sprintf("Removing IP %s (managed by this resource)", entry.Ip.ValueString()))
		}
	}

	var newIPsList []string
	for _, entry := range newList {
		newIPsList = append(newIPsList, entry.Ip.ValueString())
	}
	tflog.Debug(ctx, fmt.Sprintf("New IP list after filtering: %v (nil: %v)", newIPsList, newList == nil))

	tflog.Debug(ctx, fmt.Sprintf("Updating instance %s with new IP list", state.InstanceID.ValueString()))
	if err := patchInstanceIPAllowList(ctx, r.akpCli, state.InstanceID.ValueString(), newList); err != nil {
		instanceMutexKV.Unlock(state.InstanceID.ValueString())
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update instance: %s", err))
		tflog.Error(ctx, fmt.Sprintf("Failed to update instance: %s", err))
		return
	}

	instanceMutexKV.Unlock(state.InstanceID.ValueString())

	if err := waitForInstanceHealth(ctx, r.akpCli, instanceName); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Instance did not become healthy: %s", err))
		return
	}

	tflog.Debug(ctx, "Verifying IP list changes were applied")
	updatedEntries, _, err := getInstanceIPAllowList(ctx, r.akpCli, state.InstanceID.ValueString())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to verify IP list changes: %s", err))
	} else {
		for _, entry := range updatedEntries {
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
		for _, entry := range updatedEntries {
			remainingIPs = append(remainingIPs, entry.Ip.ValueString())
		}
		tflog.Debug(ctx, fmt.Sprintf("Verified deletion successful. Remaining IPs in instance: %v", remainingIPs))
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully deleted IP allow list for instance %s", state.InstanceID.ValueString()))
}

func (r *AkpInstanceIPAllowListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx = r.AuthCtx(ctx)

	entries, _, err := getInstanceIPAllowList(ctx, r.akpCli, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to get instance: %s", err))
		return
	}

	if entries == nil {
		entries = []*types.IPAllowListEntry{}
	}

	state := IPAllowListResourceModel{
		ID:         tftypes.StringValue(uuid.New().String()),
		InstanceID: tftypes.StringValue(req.ID),
		Entries:    entries,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func getInstanceIPAllowList(ctx context.Context, cli *AkpCli, instanceID string) ([]*types.IPAllowListEntry, string, error) {
	getResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
		return cli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
			OrganizationId: cli.OrgId,
			IdType:         idv1.Type_ID,
			Id:             instanceID,
		})
	}, "GetInstance")
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to get instance")
	}

	var entries []*types.IPAllowListEntry
	for _, e := range getResp.Instance.GetSpec().GetIpAllowList() {
		if e == nil {
			continue
		}
		entries = append(entries, &types.IPAllowListEntry{
			Ip:          tftypes.StringValue(e.GetIp()),
			Description: tftypes.StringValue(e.GetDescription()),
		})
	}
	return entries, getResp.Instance.GetName(), nil
}

func patchInstanceIPAllowList(ctx context.Context, cli *AkpCli, instanceID string, entries []*types.IPAllowListEntry) error {
	ipAllowList := make([]any, 0, len(entries))
	for _, entry := range entries {
		e := map[string]any{
			"ip": entry.Ip.ValueString(),
		}
		if !entry.Description.IsNull() && !entry.Description.IsUnknown() && entry.Description.ValueString() != "" {
			e["description"] = entry.Description.ValueString()
		}
		ipAllowList = append(ipAllowList, e)
	}

	patchStruct, err := structpb.NewStruct(map[string]any{
		"spec": map[string]any{
			"ipAllowList": ipAllowList,
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to build patch struct")
	}

	_, err = retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.PatchInstanceResponse, error) {
		return cli.Cli.PatchInstance(ctx, &argocdv1.PatchInstanceRequest{
			OrganizationId: cli.OrgId,
			Id:             instanceID,
			Patch:          patchStruct,
		})
	}, "PatchInstance")
	if err != nil {
		return errors.Wrap(err, "unable to patch instance IP allow list")
	}
	return nil
}

func waitForInstanceHealth(ctx context.Context, cli *AkpCli, instanceName string) error {
	getResourceFunc := func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
		return retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
			return cli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
				OrganizationId: cli.OrgId,
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

package akp

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/protobuf/types/known/structpb"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

var (
	_ resource.Resource                = &AkpInstanceIPAllowListResource{}
	_ resource.ResourceWithImportState = &AkpInstanceIPAllowListResource{}

	// instanceLocks serialises allow-list patches per instance so several
	// resources managing the same instance do not overwrite each other.
	instanceLocks sync.Map
)

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
	checkDuplicateIPs(&resp.Diagnostics, plan.Entries)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = r.AuthCtx(ctx)
	instanceName := r.mutateIPAllowList(ctx, &resp.Diagnostics, plan.InstanceID.ValueString(), func(current []*types.IPAllowListEntry) []*types.IPAllowListEntry {
		currentIPs := ipSet(current)
		for _, entry := range plan.Entries {
			if currentIPs[entry.Ip.ValueString()] {
				addForeignIPError(&resp.Diagnostics, entry.Ip.ValueString())
				return nil
			}
		}
		return append(current, plan.Entries...)
	})
	if resp.Diagnostics.HasError() {
		return
	}

	if err := waitForInstanceHealth(ctx, r.akpCli, instanceName); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Instance did not become healthy: %s", err))
		return
	}

	plan.ID = tftypes.StringValue(rand.Text())

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

	managedIPs := ipSet(data.Entries)
	updatedEntries := []*types.IPAllowListEntry{}
	for _, entry := range currentEntries {
		if managedIPs[entry.Ip.ValueString()] {
			updatedEntries = append(updatedEntries, entry)
		}
	}

	if len(updatedEntries) < len(data.Entries) {
		tflog.Warn(ctx, fmt.Sprintf("Some IPs managed by this resource were deleted externally. Expected %d, found %d", len(data.Entries), len(updatedEntries)))
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
	checkDuplicateIPs(&resp.Diagnostics, plan.Entries)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = r.AuthCtx(ctx)
	oldIPs := ipSet(state.Entries)
	planByIP := make(map[string]*types.IPAllowListEntry, len(plan.Entries))
	for _, entry := range plan.Entries {
		planByIP[entry.Ip.ValueString()] = entry
	}
	instanceName := r.mutateIPAllowList(ctx, &resp.Diagnostics, plan.InstanceID.ValueString(), func(current []*types.IPAllowListEntry) []*types.IPAllowListEntry {
		newList := []*types.IPAllowListEntry{}
		for _, entry := range current {
			ip := entry.Ip.ValueString()
			planned, inPlan := planByIP[ip]
			switch {
			case oldIPs[ip] && !inPlan: // removed by this resource
			case oldIPs[ip]: // managed here: the planned entry carries any description change
				newList = append(newList, planned)
			case inPlan: // about to be added, but someone else already owns it
				addForeignIPError(&resp.Diagnostics, ip)
				return nil
			default:
				newList = append(newList, entry)
			}
		}
		for _, entry := range plan.Entries {
			if !oldIPs[entry.Ip.ValueString()] {
				newList = append(newList, entry)
			}
		}
		return newList
	})
	if resp.Diagnostics.HasError() {
		return
	}

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
	managedIPs := ipSet(state.Entries)
	instanceName := r.mutateIPAllowList(ctx, &resp.Diagnostics, state.InstanceID.ValueString(), func(current []*types.IPAllowListEntry) []*types.IPAllowListEntry {
		newList := []*types.IPAllowListEntry{}
		for _, entry := range current {
			if !managedIPs[entry.Ip.ValueString()] {
				newList = append(newList, entry)
			}
		}
		return newList
	})
	if resp.Diagnostics.HasError() {
		return
	}

	if err := waitForInstanceHealth(ctx, r.akpCli, instanceName); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Instance did not become healthy: %s", err))
		return
	}

	updatedEntries, _, err := getInstanceIPAllowList(ctx, r.akpCli, state.InstanceID.ValueString())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to verify IP list changes: %s", err))
		return
	}
	for _, entry := range updatedEntries {
		if managedIPs[entry.Ip.ValueString()] {
			resp.Diagnostics.AddError(
				"Delete Verification Failed",
				fmt.Sprintf("IP %s was not successfully removed from the instance", entry.Ip.ValueString()),
			)
			return
		}
	}
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
		ID:         tftypes.StringValue(rand.Text()),
		InstanceID: tftypes.StringValue(req.ID),
		Entries:    entries,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// mutateIPAllowList fetches the instance's allow list under the per-instance
// lock, patches it with the result of mutate, and returns the instance name for
// the health wait. mutate reports failures through diags and returns nil.
func (r *AkpInstanceIPAllowListResource) mutateIPAllowList(ctx context.Context, diags *diag.Diagnostics, instanceID string, mutate func(current []*types.IPAllowListEntry) []*types.IPAllowListEntry) string {
	mu, _ := instanceLocks.LoadOrStore(instanceID, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()

	current, instanceName, err := getInstanceIPAllowList(ctx, r.akpCli, instanceID)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to get instance: %s", err))
		return ""
	}
	newList := mutate(current)
	if diags.HasError() {
		return ""
	}
	if err := patchInstanceIPAllowList(ctx, r.akpCli, instanceID, newList); err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to update instance: %s", err))
		return ""
	}
	return instanceName
}

func ipSet(entries []*types.IPAllowListEntry) map[string]bool {
	set := make(map[string]bool, len(entries))
	for _, entry := range entries {
		set[entry.Ip.ValueString()] = true
	}
	return set
}

func checkDuplicateIPs(diags *diag.Diagnostics, entries []*types.IPAllowListEntry) {
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		ip := entry.Ip.ValueString()
		if seen[ip] {
			diags.AddError("Duplicate IP", fmt.Sprintf("IP %s appears multiple times in the entries list", ip))
			return
		}
		seen[ip] = true
	}
}

func addForeignIPError(diags *diag.Diagnostics, ip string) {
	diags.AddError("Duplicate IP", fmt.Sprintf("IP %s already exists in the allow list. It may be managed by another resource", ip))
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
		return nil, "", fmt.Errorf("failed to get instance: %w", err)
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
		return fmt.Errorf("failed to build patch struct: %w", err)
	}

	_, err = retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.PatchInstanceResponse, error) {
		return cli.Cli.PatchInstance(ctx, &argocdv1.PatchInstanceRequest{
			OrganizationId: cli.OrgId,
			Id:             instanceID,
			Patch:          patchStruct,
		})
	}, "PatchInstance")
	if err != nil {
		return fmt.Errorf("unable to patch instance IP allow list: %w", err)
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
		return fmt.Errorf("instance '%s' did not become healthy: %w", instanceName, healthErr)
	}

	return nil
}

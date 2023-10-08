package akp

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/protobuf/types/known/structpb"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	argocdv1alpha1 "github.com/akuity/terraform-provider-akp/akp/apis/argocd/v1alpha1"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &AkpConfigManagementPluginResource{}
var _ resource.ResourceWithImportState = &AkpConfigManagementPluginResource{}

func NewAkpConfigManagementPluginResource() resource.Resource {
	return &AkpConfigManagementPluginResource{}
}

// AkpConfigManagementPluginResource defines the resource implementation.
type AkpConfigManagementPluginResource struct {
	akpCli *AkpCli
}

func (r *AkpConfigManagementPluginResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_management_plugin"
}

func (r *AkpConfigManagementPluginResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *AkpConfigManagementPluginResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating a ConfigManagementPlugin")
	var plan types.ConfigManagementPlugin

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("creating plan: %s", plan))

	r.upsert(ctx, &resp.Diagnostics, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AkpConfigManagementPluginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading a ConfigManagementPlugin")
	var data types.ConfigManagementPlugin
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("reading plan: %s", data))

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	refreshCMPState(ctx, &resp.Diagnostics, r.akpCli.Cli, &data, r.akpCli.OrgId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AkpConfigManagementPluginResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Updating a ConfigManagementPlugin")
	var plan types.ConfigManagementPlugin

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("updating plan: %s", plan))

	r.upsert(ctx, &resp.Diagnostics, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AkpConfigManagementPluginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Deleting a ConfigManagementPlugin")
	var plan types.ConfigManagementPlugin
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	exportReq := &argocdv1.ExportInstanceRequest{
		OrganizationId: r.akpCli.OrgId,
		IdType:         idv1.Type_ID,
		Id:             plan.InstanceID.ValueString(),
	}
	tflog.Debug(ctx, fmt.Sprintf("Export instance request: %s", exportReq))
	exportResp, err := r.akpCli.Cli.ExportInstance(ctx, exportReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to export Argo CD instance. %s", err))
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Export instance response: %s", exportResp))
	var newCMPs []*structpb.Struct
	for _, plugin := range exportResp.ConfigManagementPlugins {
		var cmp *argocdv1alpha1.ConfigManagementPlugin
		err := marshal.RemarshalTo(plugin.AsMap(), &cmp)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get ConfigManagementPlugin. %s", err))
			return
		}
		// Filter out the deleted cmp
		if cmp.Name == plan.Name.ValueString() {
			continue
		}
		newCMPs = append(newCMPs, plugin)
	}
	applyReq := &argocdv1.ApplyInstanceRequest{
		OrganizationId:          r.akpCli.OrgId,
		IdType:                  idv1.Type_ID,
		Id:                      plan.InstanceID.ValueString(),
		ConfigManagementPlugins: newCMPs,
		PruneResourceTypes:      []argocdv1.PruneResourceType{argocdv1.PruneResourceType_PRUNE_RESOURCE_TYPE_CONFIG_MANAGEMENT_PLUGINS},
	}
	tflog.Debug(ctx, fmt.Sprintf("Apply instance request: %s", applyReq))
	_, err = r.akpCli.Cli.ApplyInstance(ctx, applyReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to apply Argo CD instance. %s", err))
	}

}

func (r *AkpConfigManagementPluginResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "/")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: instance_id/name. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), idParts[1])...)
}

func (r *AkpConfigManagementPluginResource) upsert(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.ConfigManagementPlugin) {
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	apiReq := buildCMPApplyRequest(ctx, diagnostics, plan, r.akpCli.OrgId)
	tflog.Debug(ctx, fmt.Sprintf("Apply cmp request: %s", apiReq))
	_, err := r.akpCli.Cli.ApplyInstance(ctx, apiReq)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to apply Argo CD instance. %s", err))
		return
	}
	refreshCMPState(ctx, diagnostics, r.akpCli.Cli, plan, r.akpCli.OrgId)
	if diagnostics.HasError() {
		return
	}
}

func buildCMPApplyRequest(ctx context.Context, diagnostics *diag.Diagnostics, cmp *types.ConfigManagementPlugin, orgId string) *argocdv1.ApplyInstanceRequest {
	applyReq := &argocdv1.ApplyInstanceRequest{
		OrganizationId:          orgId,
		IdType:                  idv1.Type_ID,
		Id:                      cmp.InstanceID.ValueString(),
		ConfigManagementPlugins: buildCMPs(ctx, diagnostics, cmp),
	}
	return applyReq
}

func buildCMPs(ctx context.Context, diagnostics *diag.Diagnostics, cmp *types.ConfigManagementPlugin) []*structpb.Struct {
	var cs []*structpb.Struct
	apiCMP := cmp.ToConfigManagementPluginAPIModel(ctx, diagnostics)
	s, err := marshal.ApiModelToPBStruct(apiCMP)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ConfigManagementPlugin. %s", err))
		return nil
	}
	cs = append(cs, s)
	return cs
}

func refreshCMPState(ctx context.Context, diagnostics *diag.Diagnostics, client argocdv1.ArgoCDServiceGatewayClient, cmp *types.ConfigManagementPlugin, orgID string) {
	exportReq := &argocdv1.ExportInstanceRequest{
		OrganizationId: orgID,
		IdType:         idv1.Type_ID,
		Id:             cmp.InstanceID.ValueString(),
	}
	tflog.Debug(ctx, fmt.Sprintf("Export instance request: %s", exportReq))
	exportResp, err := client.ExportInstance(ctx, exportReq)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to export Argo CD instance. %s", err))
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Export instance response: %s", exportResp))
	for _, plugin := range exportResp.ConfigManagementPlugins {
		var apiCMP *argocdv1alpha1.ConfigManagementPlugin
		err = marshal.RemarshalTo(plugin.AsMap(), &apiCMP)
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get ConfigManagementPlugin. %s", err))
			return
		}
		if cmp.Name.ValueString() == apiCMP.Name {
			cmp.Update(ctx, diagnostics, apiCMP)
			return
		}
	}
}

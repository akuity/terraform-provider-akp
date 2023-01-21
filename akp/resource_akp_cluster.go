package akp

import (
	"context"
	"fmt"
	"time"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	reconv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/reconciliation/v1"
	ctxutil "github.com/akuity/api-client-go/pkg/utils/context"
	akptypes "github.com/akuity/terraform-provider-akp/akp/types"
	status "google.golang.org/grpc/status"
	codes "google.golang.org/grpc/codes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/exp/slices"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &AkpClusterResource{}
var _ resource.ResourceWithImportState = &AkpClusterResource{}

func NewAkpClusterResource() resource.Resource {
	return &AkpClusterResource{}
}

// AkpClusterResource defines the resource implementation.
type AkpClusterResource struct {
	akpCli *AkpCli
}

func (r *AkpClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *AkpClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AkpClusterResource) GetManifests(ctx context.Context, instanceId string, clusterId string) (manifests string, err error) {

	tflog.Info(ctx, "Retrieving manifests...")

	apiReq := &argocdv1.GetInstanceClusterManifestsRequest{
		OrganizationId: r.akpCli.OrgId,
		InstanceId:     instanceId,
		Id:             clusterId,
	}
	tflog.Debug(ctx, fmt.Sprintf("apiReq: %s", apiReq))
	apiResp, err := r.akpCli.Cli.GetInstanceClusterManifests(ctx, apiReq)
	if err != nil {
		return "", err
	}

	return string(apiResp.GetData()), nil
}

func (r *AkpClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan *akptypes.AkpCluster

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}
	ctx = ctxutil.SetClientCredential(ctx, r.akpCli.Cred)
	customArgoproj := plan.CustomImageRegistryArgoproj.ValueString()
	customAkuity := plan.CustomImageRegistryAkuity.ValueString()
	autoupgrade := plan.AutoUpgradeDisabled.ValueBool()
	var labels map[string]string
	var annotations map[string]string
	resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, true)...)
	resp.Diagnostics.Append(plan.Annotations.ElementsAs(ctx, &annotations, true)...)
	apiReq := &argocdv1.CreateInstanceClusterRequest{
		OrganizationId:  r.akpCli.OrgId,
		Name:            plan.Name.ValueString(),
		InstanceId:      plan.InstanceId.ValueString(),
		Description:     plan.Description.ValueString(),
		Namespace:       plan.Namespace.ValueString(),
		NamespaceScoped: plan.NamespaceScoped.ValueBool(),
		Data: &argocdv1.ClusterData{
			Size: akptypes.StringClusterSize[plan.Size.ValueString()],
			CustomImageRegistryArgoproj: &customArgoproj,
			CustomImageRegistryAkuity:   &customAkuity,
			AutoUpgradeDisabled:         &autoupgrade,
			Labels:                      labels,
			Annotations:                 annotations,
		},
		Upsert: false,
	}
	tflog.Info(ctx, "Creating Cluster...")
	tflog.Debug(ctx, fmt.Sprintf("Api Request: %s", apiReq))
	apiResp, err := r.akpCli.Cli.CreateInstanceCluster(ctx, apiReq)
	tflog.Debug(ctx, fmt.Sprintf("Api Response: %s", apiResp))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Akuity cluster. %s", err))
		return
	}

	cluster := apiResp.GetCluster()
	reconStatus := cluster.GetReconciliationStatus()
	breakStatusesRecon := []reconv1.StatusCode{reconv1.StatusCode_STATUS_CODE_SUCCESSFUL, reconv1.StatusCode_STATUS_CODE_FAILED}
	for !slices.Contains(breakStatusesRecon, reconStatus.GetCode()) {
		time.Sleep(1 * time.Second)
		apiResp2, err := r.akpCli.Cli.GetInstanceCluster(ctx, &argocdv1.GetInstanceClusterRequest{
			OrganizationId: r.akpCli.OrgId,
			InstanceId:     plan.InstanceId.ValueString(),
			Id:             cluster.GetId(),
			IdType:         idv1.Type_ID,
		})
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to check health of cluster. %s", err))
			return
		}
		cluster = apiResp2.GetCluster()
		reconStatus = cluster.GetReconciliationStatus()
		tflog.Debug(ctx, fmt.Sprintf("Cluster instance status: %s", reconStatus.String()))
	}
	tflog.Info(ctx, "Cluster created")

	protoCluster := &akptypes.ProtoCluster{Cluster: cluster}
	state, diag := protoCluster.FromProto(plan.InstanceId.ValueString())
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
	}
	manifests, err := r.GetManifests(ctx, plan.InstanceId.ValueString(), cluster.GetId())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get cluster manifests. %s", err))
		return
	}

	state.Manifests = types.StringValue(manifests)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AkpClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state *akptypes.AkpCluster

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = ctxutil.SetClientCredential(ctx, r.akpCli.Cred)
	apiReq := &argocdv1.GetInstanceClusterRequest{
		OrganizationId: r.akpCli.OrgId,
		InstanceId:     state.InstanceId.ValueString(),
		Id:             state.Id.ValueString(),
		IdType:         idv1.Type_ID,
	}
	tflog.Debug(ctx, fmt.Sprintf("Api Request: %s", apiReq))
	apiResp, err := r.akpCli.Cli.GetInstanceCluster(ctx, apiReq)
	switch status.Code(err) {
	case codes.OK:
		tflog.Debug(ctx, fmt.Sprintf("Api Response: %s", apiResp))
	case codes.NotFound:
		resp.State.RemoveResource(ctx)
		return
	default:
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Akuity cluster. %s", err))
		return
	}

	resp.Diagnostics.Append(state.UpdateFromProto(apiResp.GetCluster())...)

	if state.Manifests.IsNull() || state.Manifests.IsUnknown() {
		manifests, _ := r.GetManifests(ctx, state.InstanceId.ValueString(), state.Id.ValueString())
		state.Manifests = types.StringValue(manifests)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AkpClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan *akptypes.AkpCluster

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = ctxutil.SetClientCredential(ctx, r.akpCli.Cred)
	customArgoproj := plan.CustomImageRegistryArgoproj.ValueString()
	customAkuity := plan.CustomImageRegistryAkuity.ValueString()
	autoupgrade := plan.AutoUpgradeDisabled.ValueBool()
	var labels map[string]string
	var annotations map[string]string
	resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, true)...)
	resp.Diagnostics.Append(plan.Annotations.ElementsAs(ctx, &annotations, true)...)
	apiReq :=&argocdv1.UpdateInstanceClusterRequest{
		OrganizationId: r.akpCli.OrgId,
		InstanceId:     plan.InstanceId.ValueString(),
		Id:             plan.Id.ValueString(),
		Description:    plan.Description.ValueString(),
		Data: &argocdv1.ClusterData{
			Size: akptypes.StringClusterSize[plan.Size.ValueString()],
			CustomImageRegistryArgoproj: &customArgoproj,
			CustomImageRegistryAkuity:   &customAkuity,
			AutoUpgradeDisabled:         &autoupgrade,
			Labels:                      labels,
			Annotations:                 annotations,
		},
	}
	tflog.Debug(ctx, fmt.Sprintf("Api Request: %s", apiReq))
	apiResp, err := r.akpCli.Cli.UpdateInstanceCluster(ctx, apiReq)
	tflog.Debug(ctx, fmt.Sprintf("Api Respons: %s", apiResp))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Akuity cluster. %s", err))
		return
	}

	cluster := apiResp.GetCluster()
	protoCluster := &akptypes.ProtoCluster{Cluster: cluster}
	// Update state
	state, diag := protoCluster.FromProto(plan.InstanceId.ValueString())
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
	}
	manifests, err := r.GetManifests(ctx, plan.InstanceId.ValueString(), cluster.GetId())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get cluster manifests. %s", err))
		return
	}

	state.Manifests = types.StringValue(manifests)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AkpClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state *akptypes.AkpCluster

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = ctxutil.SetClientCredential(ctx, r.akpCli.Cred)
	apiReq :=&argocdv1.DeleteInstanceClusterRequest{
		OrganizationId: r.akpCli.OrgId,
		InstanceId:     state.InstanceId.ValueString(),
		Id:             state.Id.ValueString(),
	}
	tflog.Debug(ctx, fmt.Sprintf("Api Request: %s", apiReq))
	apiResp, err := r.akpCli.Cli.DeleteInstanceCluster(ctx, apiReq)
	tflog.Debug(ctx, fmt.Sprintf("Api Response: %s", apiResp))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Akuity cluster. %s", err))
		return
	}
	time.Sleep(1 * time.Second)
}

// TODO: Implement cluster import
func (r *AkpClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

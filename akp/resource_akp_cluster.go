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

	apiReq := &argocdv1.GetOrganizationInstanceClusterManifestsRequest{
		OrganizationId: r.akpCli.OrgId,
		InstanceId:     instanceId,
		Id:             clusterId,
	}
	tflog.Debug(ctx, fmt.Sprintf("apiReq: %s", apiReq))
	apiResp, err := r.akpCli.Cli.GetOrganizationInstanceClusterManifests(ctx, apiReq)
	if err != nil {
		return "", err
	}

	return apiResp.GetManifests(), nil
}

func (r *AkpClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan *akptypes.AkpCluster

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}
	ctx = ctxutil.SetClientCredential(ctx, r.akpCli.Cred)
	apiReq := &argocdv1.CreateOrganizationInstanceClusterRequest{
		OrganizationId:  r.akpCli.OrgId,
		Name:            plan.Name.ValueString(),
		InstanceId:      plan.InstanceId.ValueString(),
		Description:     plan.Description.ValueString(),
		Namespace:       plan.Namespace.ValueString(),
		NamespaceScoped: plan.NamespaceScoped.ValueBool(),
		Data: &argocdv1.ClusterData{
			Size: argocdv1.ClusterSize_CLUSTER_SIZE_SMALL,
		},
		Upsert: false,
	}
	tflog.Info(ctx, "Creating Cluster...")
	tflog.Debug(ctx, fmt.Sprintf("apiReq: %s", apiReq))
	apiResp, err := r.akpCli.Cli.CreateOrganizationInstanceCluster(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Akuity cluster. %s", err))
		return
	}

	cluster := apiResp.GetCluster()
	reconStatus := cluster.GetReconciliationStatus()
	breakStatusesRecon := []reconv1.StatusCode{reconv1.StatusCode_STATUS_CODE_SUCCESSFUL, reconv1.StatusCode_STATUS_CODE_FAILED}
	for !slices.Contains(breakStatusesRecon, reconStatus.GetCode()) {
		time.Sleep(2 * time.Second)
		apiResp2, err := r.akpCli.Cli.GetOrganizationInstanceCluster(ctx, &argocdv1.GetOrganizationInstanceClusterRequest{
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
	state := protoCluster.FromProto(plan.InstanceId.ValueString())

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
	apiResp, err := r.akpCli.Cli.GetOrganizationInstanceCluster(ctx, &argocdv1.GetOrganizationInstanceClusterRequest{
		OrganizationId: r.akpCli.OrgId,
		InstanceId:     state.InstanceId.ValueString(),
		Id:             state.Id.ValueString(),
		IdType:         idv1.Type_ID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Akuity cluster. %s", err))
		return
	}

	// Update state
	state.UpdateFromProto(apiResp.GetCluster())

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
	apiResp, err := r.akpCli.Cli.UpdateOrganizationInstanceCluster(ctx, &argocdv1.UpdateOrganizationInstanceClusterRequest{
		OrganizationId: r.akpCli.OrgId,
		InstanceId:     plan.InstanceId.ValueString(),
		Id:             plan.Id.ValueString(),
		Description:    plan.Description.ValueString(),
		Data: &argocdv1.ClusterData{
			Size: argocdv1.ClusterSize_CLUSTER_SIZE_SMALL,
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Akuity cluster. %s", err))
		return
	}

	cluster := apiResp.GetCluster()
	protoCluster := &akptypes.ProtoCluster{Cluster: cluster}
	// Update state
	state := protoCluster.FromProto(plan.InstanceId.ValueString())

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
	_, err := r.akpCli.Cli.DeleteOrganizationInstanceCluster(ctx, &argocdv1.DeleteOrganizationInstanceClusterRequest{
		OrganizationId: r.akpCli.OrgId,
		InstanceId:     state.InstanceId.ValueString(),
		Id:             state.Id.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Akuity cluster. %s", err))
		return
	}
}

// TODO: Implement cluster import
func (r *AkpClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

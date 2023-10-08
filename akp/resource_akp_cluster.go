package akp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/client-go/rest"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	reconv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/reconciliation/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/kube"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
	"github.com/akuity/terraform-provider-akp/akp/types"
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

func (r *AkpClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating a Cluster")
	var plan types.Cluster

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.upsert(ctx, &resp.Diagnostics, &plan, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AkpClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading a Cluster")
	var data types.Cluster
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	refreshClusterState(ctx, &resp.Diagnostics, r.akpCli.Cli, &data, r.akpCli.OrgId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AkpClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Updating a Cluster")
	var plan types.Cluster

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.upsert(ctx, &resp.Diagnostics, &plan, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AkpClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Deleting a Cluster")
	var plan types.Cluster
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	kubeconfig, diag := getKubeconfig(plan.Kubeconfig)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the manifests
	if kubeconfig != nil && plan.RemoveAgentResourcesOnDestroy.ValueBool() {
		resp.Diagnostics.Append(deleteManifests(ctx, getManifests(ctx, &resp.Diagnostics, r.akpCli.Cli, r.akpCli.OrgId, &plan), kubeconfig)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	apiReq := &argocdv1.DeleteInstanceClusterRequest{
		OrganizationId: r.akpCli.OrgId,
		InstanceId:     plan.InstanceID.ValueString(),
		Id:             plan.ID.ValueString(),
	}
	_, err := r.akpCli.Cli.DeleteInstanceCluster(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Akuity cluster. %s", err))
		return
	}
	// Give it some time to remove the cluster. This is useful when the terraform provider is performing a replace operation, to give it enough time to destroy the previous cluster.
	time.Sleep(2 * time.Second)
}

func (r *AkpClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func (r *AkpClusterResource) upsert(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.Cluster, isCreate bool) {
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	apiReq := buildClusterApplyRequest(ctx, diagnostics, plan, r.akpCli.OrgId)
	tflog.Debug(ctx, fmt.Sprintf("Apply cluster request: %s", apiReq))
	_, err := r.akpCli.Cli.ApplyInstance(ctx, apiReq)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
		return
	}
	refreshClusterState(ctx, diagnostics, r.akpCli.Cli, plan, r.akpCli.OrgId)
	if diagnostics.HasError() {
		return
	}

	// Apply agent manifests to clusters if the kubeconfig is specified for cluster.
	kubeconfig, diag := getKubeconfig(plan.Kubeconfig)
	diagnostics.Append(diag...)
	if diagnostics.HasError() {
		return
	}

	// Apply the manifests
	if kubeconfig != nil && isCreate {
		diagnostics.Append(applyManifests(ctx, getManifests(ctx, diagnostics, r.akpCli.Cli, r.akpCli.OrgId, plan), kubeconfig)...)
		if diagnostics.HasError() {
			return
		}
		if err := waitClusterHealthStatus(ctx, r.akpCli.Cli, r.akpCli.OrgId, plan); err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("unable to create Argo CD instance: %s", err))
		}
	}
}

func refreshClusterState(ctx context.Context, diagnostics *diag.Diagnostics, client argocdv1.ArgoCDServiceGatewayClient, cluster *types.Cluster, orgID string) {
	clusterReq := &argocdv1.GetInstanceClusterRequest{
		OrganizationId: orgID,
		InstanceId:     cluster.InstanceID.ValueString(),
		Id:             cluster.Name.ValueString(),
		IdType:         idv1.Type_NAME,
	}

	tflog.Debug(ctx, fmt.Sprintf("Get cluster request: %s", clusterReq))
	clusterResp, err := client.GetInstanceCluster(ctx, clusterReq)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Argo CD cluster: %s", err))
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Get cluster response: %s", clusterResp))
	cluster.Update(ctx, diagnostics, clusterResp.GetCluster())
}

func buildClusterApplyRequest(ctx context.Context, diagnostics *diag.Diagnostics, cluster *types.Cluster, orgId string) *argocdv1.ApplyInstanceRequest {
	applyReq := &argocdv1.ApplyInstanceRequest{
		OrganizationId: orgId,
		IdType:         idv1.Type_ID,
		Id:             cluster.InstanceID.ValueString(),
		Clusters:       buildClusters(ctx, diagnostics, cluster),
	}
	return applyReq
}

func buildClusters(ctx context.Context, diagnostics *diag.Diagnostics, cluster *types.Cluster) []*structpb.Struct {
	var cs []*structpb.Struct
	apiCluster := cluster.ToClusterAPIModel(ctx, diagnostics)
	s, err := marshal.ApiModelToPBStruct(apiCluster)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Cluster. %s", err))
		return nil
	}
	cs = append(cs, s)
	return cs
}

func getKubeconfig(kubeConfig *types.Kubeconfig) (*rest.Config, diag.Diagnostics) {
	var diag diag.Diagnostics
	if kubeConfig == nil {
		return nil, diag
	}
	kcfg, err := kube.InitializeConfiguration(kubeConfig)
	if err != nil {
		diag.AddError("Kubectl error", fmt.Sprintf("Cannot insitialize Kubectl. Check kubernetes configuration. Error: %s", err))
		return nil, diag
	}
	return kcfg, diag
}

func getManifests(ctx context.Context, diagnostics *diag.Diagnostics, client argocdv1.ArgoCDServiceGatewayClient, orgId string, cluster *types.Cluster) string {
	clusterReq := &argocdv1.GetInstanceClusterRequest{
		OrganizationId: orgId,
		InstanceId:     cluster.InstanceID.ValueString(),
		Id:             cluster.Name.ValueString(),
		IdType:         idv1.Type_NAME,
	}
	clusterResp, err := client.GetInstanceCluster(ctx, clusterReq)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance clusters, got error: %s", err))
		return ""
	}
	c, err := waitClusterReconStatus(ctx, client, clusterResp.GetCluster(), orgId, cluster.ID.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to check cluster reconciliation status. %s", err))
		return ""
	}
	apiReq := &argocdv1.GetInstanceClusterManifestsRequest{
		OrganizationId: orgId,
		InstanceId:     cluster.InstanceID.ValueString(),
		Id:             c.Id,
	}
	apiResp, err := client.GetInstanceClusterManifests(ctx, apiReq)
	if err != nil {
		diagnostics.AddError("Akuity API error", fmt.Sprintf("Unable to download manifests: %s", err))
		return ""
	}
	return string(apiResp.GetData())
}

func applyManifests(ctx context.Context, manifests string, cfg *rest.Config) diag.Diagnostics {
	diag := diag.Diagnostics{}
	kubectl, err := kube.NewKubectl(cfg)
	if err != nil {
		diag.AddError("Kubernetes error", fmt.Sprintf("failed to create kubectl, error=%s", err))
	}
	resources, err := kube.SplitYAML([]byte(manifests))
	if err != nil {
		diag.AddError("YAML error", fmt.Sprintf("failed to parse manifest, error=%s", err))
	}

	for _, un := range resources {
		msg, err := kubectl.ApplyResource(ctx, &un, kube.ApplyOpts{})
		if err != nil {
			diag.AddError("Kubernetes error", fmt.Sprintf("failed to apply manifest %s, error=%s", un, err))
			return diag
		}
		tflog.Debug(ctx, msg)
	}
	return diag
}

func deleteManifests(ctx context.Context, manifests string, cfg *rest.Config) diag.Diagnostics {
	diag := diag.Diagnostics{}
	kubectl, err := kube.NewKubectl(cfg)
	if err != nil {
		diag.AddError("Kubernetes error", fmt.Sprintf("failed to create kubectl, error=%s", err))
	}
	resources, err := kube.SplitYAML([]byte(manifests))
	tflog.Info(ctx, fmt.Sprintf("%d resources to delete", len(resources)))
	if err != nil {
		diag.AddError("YAML error", fmt.Sprintf("failed to parse manifest, error=%s", err))
	}

	// Delete the resources in reverse order
	for i := len(resources) - 1; i >= 0; i-- {
		msg, err := kubectl.DeleteResource(ctx, &resources[i], kube.DeleteOpts{
			IgnoreNotFound:  true,
			WaitForDeletion: true,
			Force:           false,
		})
		if err != nil {
			diag.AddError("Kubernetes error", fmt.Sprintf("failed to delete manifest %s, error=%s", &resources[i], err))
			return diag
		}
		tflog.Debug(ctx, msg)
	}
	return diag
}

func waitClusterHealthStatus(ctx context.Context, client argocdv1.ArgoCDServiceGatewayClient, orgID string, c *types.Cluster) error {
	cluster := &argocdv1.Cluster{}
	healthStatus := cluster.GetHealthStatus()
	breakStatusesHealth := []healthv1.StatusCode{healthv1.StatusCode_STATUS_CODE_HEALTHY, healthv1.StatusCode_STATUS_CODE_DEGRADED}

	for !slices.Contains(breakStatusesHealth, healthStatus.GetCode()) {
		time.Sleep(1 * time.Second)
		apiResp, err := client.GetInstanceCluster(ctx, &argocdv1.GetInstanceClusterRequest{
			OrganizationId: orgID,
			InstanceId:     c.InstanceID.ValueString(),
			Id:             c.Name.ValueString(),
			IdType:         idv1.Type_NAME,
		})
		if err != nil {
			return err
		}
		cluster = apiResp.GetCluster()
		healthStatus = cluster.GetHealthStatus()
		tflog.Debug(ctx, fmt.Sprintf("Cluster health status: %s", healthStatus.String()))
	}
	return nil
}

func waitClusterReconStatus(ctx context.Context, client argocdv1.ArgoCDServiceGatewayClient, cluster *argocdv1.Cluster, orgId, instanceId string) (*argocdv1.Cluster, error) {
	reconStatus := cluster.GetReconciliationStatus()
	breakStatusesRecon := []reconv1.StatusCode{reconv1.StatusCode_STATUS_CODE_SUCCESSFUL, reconv1.StatusCode_STATUS_CODE_FAILED}

	for !slices.Contains(breakStatusesRecon, reconStatus.GetCode()) {
		time.Sleep(1 * time.Second)
		apiResp, err := client.GetInstanceCluster(ctx, &argocdv1.GetInstanceClusterRequest{
			OrganizationId: orgId,
			InstanceId:     instanceId,
			Id:             cluster.Id,
			IdType:         idv1.Type_ID,
		})
		if err != nil {
			return nil, err
		}
		cluster = apiResp.GetCluster()
		reconStatus = cluster.GetReconciliationStatus()
		tflog.Debug(ctx, fmt.Sprintf("Cluster recon status: %s", reconStatus.String()))
	}
	return cluster, nil
}

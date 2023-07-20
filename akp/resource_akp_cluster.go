package akp

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

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
	tflog.Debug(ctx, "Creating an AKP ArgoCD Instance")
	var plan types.Cluster

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.upsert(ctx, &resp.Diagnostics, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AkpClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading an AKP Cluster")

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
	var plan types.Cluster

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.upsert(ctx, &resp.Diagnostics, &plan)
	tflog.Info(ctx, fmt.Sprintf("-------------update:%+v", plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AkpClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Deleting an Cluster")
	var plan types.Cluster
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	kubeconfig, diag := getKubeconfig(ctx, plan.Kubeconfig)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the manifests
	if kubeconfig != nil {
		tflog.Info(ctx, "Deleting the manifests...")
		resp.Diagnostics.Append(deleteManifests(ctx, plan.Manifests.ValueString(), kubeconfig)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	apiReq := &argocdv1.DeleteInstanceClusterRequest{
		OrganizationId: r.akpCli.OrgId,
		InstanceId:     plan.InstanceID.ValueString(),
		Id:             plan.ID.ValueString(),
	}
	tflog.Debug(ctx, fmt.Sprintf("Api Request: %s", apiReq))
	apiResp, err := r.akpCli.Cli.DeleteInstanceCluster(ctx, apiReq)
	tflog.Debug(ctx, fmt.Sprintf("Api Response: %s", apiResp))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Akuity cluster. %s", err))
		return
	}
}

func (r *AkpClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *AkpClusterResource) upsert(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.Cluster) {
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	apiReq := buildClusterApplyRequest(ctx, diagnostics, plan, r.akpCli.OrgId)
	tflog.Info(ctx, fmt.Sprintf("----------cluste api req:%+v", apiReq))
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
	kubeconfig, diag := getKubeconfig(ctx, plan.Kubeconfig)
	diagnostics.Append(diag...)
	if diagnostics.HasError() {
		return
	}

	// Apply the manifests
	if kubeconfig != nil {
		tflog.Info(ctx, "Applying the manifests...")
		diagnostics.Append(applyManifests(ctx, plan.Manifests.ValueString(), kubeconfig)...)
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

	tflog.Debug(ctx, fmt.Sprintf("API Req: %s", clusterReq))
	clusterResp, err := client.GetInstanceCluster(ctx, clusterReq)
	tflog.Debug(ctx, fmt.Sprintf("API Resp: %s", clusterResp))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Argo CD instance: %s", err))
		return
	}
	apiCluster := clusterResp.GetCluster()
	cluster.ID = tftypes.StringValue(apiCluster.GetId())
	cluster.Name = tftypes.StringValue(apiCluster.GetName())
	cluster.Namespace = tftypes.StringValue(apiCluster.GetNamespace())
	labels, d := tftypes.MapValueFrom(ctx, tftypes.StringType, apiCluster.GetData().GetLabels())
	if d.HasError() {
		labels = tftypes.MapNull(tftypes.StringType)
	}
	diagnostics.Append(d...)
	annotations, d := tftypes.MapValueFrom(ctx, tftypes.StringType, apiCluster.GetData().GetAnnotations())
	if d.HasError() {
		annotations = tftypes.MapNull(tftypes.StringType)
	}
	diagnostics.Append(d...)
	jsonData, err := apiCluster.GetData().GetKustomization().MarshalJSON()
	if err != nil {
		diagnostics.AddError("getting cluster kustomization", fmt.Sprintf("%s", err.Error()))
	}
	yamlData, err := yaml.JSONToYAML(jsonData)
	if err != nil {
		diagnostics.AddError("getting cluster kustomization", fmt.Sprintf("%s", err.Error()))
	}

	kustomization := tftypes.StringValue(string(yamlData))
	if cluster.Spec != nil {
		rawPlan := runtime.RawExtension{}
		old := cluster.Spec.Data.Kustomization
		if err := yaml.Unmarshal([]byte(old.ValueString()), &rawPlan); err != nil {
			diagnostics.AddError("failed unmarshal kustomization string to yaml", err.Error())
		}

		oldYamlData, err := yaml.Marshal(&rawPlan)
		if err != nil {
			diagnostics.AddError("failed to convert json to yaml data", err.Error())
		}
		if bytes.Equal(oldYamlData, yamlData) {
			kustomization = old
		}
	}

	cluster.Labels = labels
	cluster.Annotations = annotations
	cluster.Spec = &types.ClusterSpec{
		Description:     tftypes.StringValue(apiCluster.GetDescription()),
		NamespaceScoped: tftypes.BoolValue(apiCluster.GetNamespaceScoped()),
		Data: types.ClusterData{
			Size:                tftypes.StringValue(types.ClusterSizeString[apiCluster.GetData().GetSize()]),
			AutoUpgradeDisabled: tftypes.BoolValue(apiCluster.GetData().GetAutoUpgradeDisabled()),
			Kustomization:       kustomization,
			AppReplication:      tftypes.BoolValue(apiCluster.GetData().GetAppReplication()),
			TargetVersion:       tftypes.StringValue(apiCluster.GetData().GetTargetVersion()),
			RedisTunneling:      tftypes.BoolValue(apiCluster.GetData().GetRedisTunneling()),
		},
	}
	cluster.Manifests = getManifests(ctx, diagnostics, client, orgID, cluster)
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
	apiCluster := types.ToClusterAPIModel(ctx, diagnostics, cluster)
	m := map[string]interface{}{}
	if err := marshal.RemarshalTo(apiCluster, &m); err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
		return nil
	}
	cs = append(cs, s)
	return cs
}

func getKubeconfig(ctx context.Context, kubeConf tftypes.Object) (*rest.Config, diag.Diagnostics) {
	var kubeConfig types.Kubeconfig
	var diag diag.Diagnostics
	if kubeConf.IsNull() || kubeConf.IsUnknown() {
		diag.AddWarning("Kubectl not configured", "Cannot update agent because kubectl configuration is missing")
		return nil, diag
	}
	tflog.Debug(ctx, fmt.Sprintf("Kube config: %s", kubeConf))
	diag = kubeConf.As(ctx, &kubeConfig, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	if diag.HasError() {
		return nil, diag
	}
	kcfg, err := kube.InitializeConfiguration(&kubeConfig)
	tflog.Debug(ctx, fmt.Sprintf("Kcfg: %s", kcfg))
	if err != nil {
		diag.AddError("Kubectl error", fmt.Sprintf("Cannot insitialize Kubectl. Check kubernetes configuration. Error: %s", err))
		return nil, diag
	}
	return kcfg, diag
}

func getManifests(ctx context.Context, diagnostics *diag.Diagnostics, client argocdv1.ArgoCDServiceGatewayClient, orgId string, cluster *types.Cluster) tftypes.String {
	clusterReq := &argocdv1.GetInstanceClusterRequest{
		OrganizationId: orgId,
		InstanceId:     cluster.InstanceID.ValueString(),
		Id:             cluster.Name.ValueString(),
		IdType:         idv1.Type_NAME,
	}
	clusterResp, err := client.GetInstanceCluster(ctx, clusterReq)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance clusters, got error: %s", err))
		return tftypes.StringNull()
	}
	c, err := waitClusterReconStatus(ctx, client, clusterResp.GetCluster(), orgId, cluster.ID.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to check cluster reconciliation status. %s", err))
		return tftypes.StringNull()
	}
	apiReq := &argocdv1.GetInstanceClusterManifestsRequest{
		OrganizationId: orgId,
		InstanceId:     cluster.InstanceID.ValueString(),
		Id:             c.Id,
	}
	apiResp, err := client.GetInstanceClusterManifests(ctx, apiReq)
	if err != nil {
		diagnostics.AddError("Akuity API error", fmt.Sprintf("Unable to download manifests: %s", err))
		return tftypes.StringNull()
	}
	return tftypes.StringValue(string(apiResp.GetData()))
}

func applyManifests(ctx context.Context, manifests string, cfg *rest.Config) diag.Diagnostics {
	diag := diag.Diagnostics{}
	kubectl, err := kube.NewKubectl(cfg)
	if err != nil {
		diag.AddError("Kubernetes error", fmt.Sprintf("failed to create kubectl, error=%s", err))
	}
	resources, err := kube.SplitYAML([]byte(manifests))
	tflog.Info(ctx, fmt.Sprintf("%d resources to create", len(resources)))
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

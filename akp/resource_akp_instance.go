package akp

import (
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
	"k8s.io/client-go/rest"

	reconv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/reconciliation/v1"
	"github.com/akuity/terraform-provider-akp/akp/marshal"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
	"github.com/akuity/terraform-provider-akp/akp/kube"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &AkpInstanceResource{}
var _ resource.ResourceWithImportState = &AkpInstanceResource{}

func NewAkpInstanceResource() resource.Resource {
	return &AkpInstanceResource{}
}

type AkpInstanceResource struct {
	akpCli *AkpCli
}

func (r *AkpInstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *AkpInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AkpInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating an AKP ArgoCD Instance")
	var plan types.Instance

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.upSert(ctx, &resp.Diagnostics, &plan)
	tflog.Info(ctx, fmt.Sprintf("-------------create:%+v", plan.ArgoCD))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AkpInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading an AKP Argo CD Instance")

	var data types.Instance
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.MaskLogStrings(ctx, data.GetSensitiveStrings()...)
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	refreshState(ctx, &resp.Diagnostics, r.akpCli.Cli, &data, r.akpCli.OrgId)
	tflog.Info(ctx, fmt.Sprintf("-------------read:%+v", data.ArgoCD))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AkpInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Updating an AKP ArgoCD Instance")
	var plan types.Instance

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.upSert(ctx, &resp.Diagnostics, &plan)
	tflog.Info(ctx, fmt.Sprintf("-------------update:%+v", plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AkpInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Deleting an Argo CD Instance")
	var state *types.Instance

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	_, err := r.akpCli.Cli.DeleteInstance(ctx, &argocdv1.DeleteInstanceRequest{
		Id:             state.ID.ValueString(),
		OrganizationId: r.akpCli.OrgId,
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Argo CD instance, got error: %s", err))
		return
	}
	tflog.Info(ctx, "Instance deleted")
}

func (r *AkpInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *AkpInstanceResource) upSert(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.Instance) {
	// Mark sensitive secret data
	tflog.MaskLogStrings(ctx, plan.GetSensitiveStrings()...)

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	apiReq := buildApplyRequest(ctx, diagnostics, plan, r.akpCli.OrgId)
	_, err := r.akpCli.Cli.ApplyInstance(ctx, apiReq)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("------apireq:%+v", apiReq))
	tflog.Info(ctx, fmt.Sprintf("------upSert:%+v", plan))

	getInstanceReq := &argocdv1.GetInstanceRequest{
		OrganizationId: r.akpCli.OrgId,
		IdType:         idv1.Type_NAME,
		Id:             apiReq.Id,
	}

	tflog.Debug(ctx, fmt.Sprintf("API Req: %s", apiReq))
	getInstanceResp, err := r.akpCli.Cli.GetInstance(ctx, getInstanceReq)
	tflog.Debug(ctx, fmt.Sprintf("API Resp: %s", getInstanceResp))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Argo CD instance: %s", err))
		return
	}
	plan.ID = tftypes.StringValue(getInstanceResp.Instance.Id)

	refreshState(ctx, diagnostics, r.akpCli.Cli, plan, r.akpCli.OrgId)
	if diagnostics.HasError() {
		return
	}

	// Apply agent manifests to clusters if the kubeconfig is specified for cluster.
	for _, cluster := range plan.Clusters {
		kubeconfig, diag := getKubeconfig(ctx, cluster.Kubeconfig)
		diagnostics.Append(diag...)
		if diagnostics.HasError() {
			return
		}
		// Apply the manifests
		if kubeconfig != nil {
			clusterReq := &argocdv1.GetInstanceClusterRequest{
				OrganizationId: r.akpCli.OrgId,
				InstanceId:     plan.ID.ValueString(),
				Id:             cluster.Name.ValueString(),
				IdType:         idv1.Type_NAME,
			}
			clusterResp, err := r.akpCli.Cli.GetInstanceCluster(ctx, clusterReq)
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance clusters, got error: %s", err))
			}
			tflog.Info(ctx, "Applying the manifests...")
			diagnostics.Append(applyManifests(ctx, cluster.Manifests.ValueString(), kubeconfig)...)
			if diagnostics.HasError() {
				return
			}
			clusterResp.Cluster, err = waitClusterHealthStatus(ctx, r.akpCli.Cli, clusterResp.Cluster, plan.ID.ValueString(), clusterResp.Cluster.Id)
		}
	}
}

func buildApplyRequest(ctx context.Context, diagnostics *diag.Diagnostics, instance *types.Instance, orgID string) *argocdv1.ApplyInstanceRequest {
	applyReq := &argocdv1.ApplyInstanceRequest{
		OrganizationId:                orgID,
		IdType:                        idv1.Type_NAME,
		Id:                            instance.ArgoCD.Name.ValueString(),
		Argocd:                        buildArgoCD(ctx, diagnostics, instance),
		ArgocdConfigmap:               buildConfigMap(ctx, diagnostics, &instance.ArgoCDConfigMap, "argocd-cm"),
		ArgocdRbacConfigmap:           buildConfigMap(ctx, diagnostics, &instance.ArgoCDRBACConfigMap, "argocd-rbac-cm"),
		ArgocdSecret:                  buildSecret(ctx, diagnostics, &instance.ArgoCDSecret, "argocd-secret"),
		NotificationsConfigmap:        buildConfigMap(ctx, diagnostics, &instance.NotificationsConfigMap, "argocd-notifications-cm"),
		NotificationsSecret:           buildSecret(ctx, diagnostics, &instance.NotificationsSecret, "argocd-notifications-secret"),
		ImageUpdaterConfigmap:         buildConfigMap(ctx, diagnostics, &instance.ImageUpdaterConfigMap, "argocd-image-updater-config"),
		ImageUpdaterSshConfigmap:      buildConfigMap(ctx, diagnostics, &instance.ImageUpdaterSSHConfigMap, "argocd-image-updater-ssh-config"),
		ImageUpdaterSecret:            buildSecret(ctx, diagnostics, &instance.ImageUpdaterSecret, "argocd-image-updater-secret"),
		Clusters:                      buildClusters(ctx, diagnostics, instance),
		ArgocdKnownHostsConfigmap:     buildConfigMap(ctx, diagnostics, &instance.ArgoCDKnownHostsConfigMap, "argocd-ssh-known-hosts-cm"),
		ArgocdTlsCertsConfigmap:       buildConfigMap(ctx, diagnostics, &instance.ArgoCDTLSCertsConfigMap, "argocd-tls-certs-cm"),
		RepoCredentialSecrets:         buildSecrets(ctx, diagnostics, instance.RepoCredentialSecrets),
		RepoTemplateCredentialSecrets: buildSecrets(ctx, diagnostics, instance.RepoTemplateCredentialSecrets),
		PruneRepoCredentialSecrets:    true,
		PruneClusters:                 true,
	}
	return applyReq
}

func buildArgoCD(ctx context.Context, diag *diag.Diagnostics, instance *types.Instance) *structpb.Struct {
	apiArgoCD := types.ToArgoCDAPIModel(ctx, diag, &instance.ArgoCD)
	argocdMap := map[string]interface{}{}
	if err := marshal.RemarshalTo(apiArgoCD, &argocdMap); err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to creat Argo CD instance. %s", err))
	}
	tflog.Info(ctx, fmt.Sprintf("------reflect:%+v", argocdMap))
	s, err := structpb.NewStruct(argocdMap)
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
		return nil
	}
	return s
}

func buildClusters(ctx context.Context, diagnostics *diag.Diagnostics, instance *types.Instance) []*structpb.Struct {
	var cs []*structpb.Struct
	for _, cluster := range instance.Clusters {
		apiCluster := types.ToClusterAPIModel(ctx, diagnostics, &cluster)
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
	}
	return cs
}

func buildSecrets(ctx context.Context, diagnostics *diag.Diagnostics, secrets []types.Secret) []*structpb.Struct {
	var res []*structpb.Struct
	for _, secret := range secrets {
		res = append(res, buildSecret(ctx, diagnostics, &secret, secret.Name.ValueString()))
	}
	return res
}

func buildConfigMap(ctx context.Context, diagnostics *diag.Diagnostics, cm *types.ConfigMap, name string) *structpb.Struct {
	apiModel := types.ToConfigMapAPIModel(ctx, diagnostics, cm, name)
	m := map[string]interface{}{}
	if err := marshal.RemarshalTo(apiModel, &m); err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
	}
	configMap, err := structpb.NewStruct(m)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
		return nil
	}
	return configMap
}

func buildSecret(ctx context.Context, diagnostics *diag.Diagnostics, secret *types.Secret, name string) *structpb.Struct {
	apiModel := types.ToSecretAPIModel(ctx, diagnostics, secret, name)
	m := map[string]interface{}{}
	if err := marshal.RemarshalTo(apiModel, &m); err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
		return nil
	}
	return s
}

func refreshState(ctx context.Context, diagnostics *diag.Diagnostics, client argocdv1.ArgoCDServiceGatewayClient, instance *types.Instance, orgID string) {
	exportResp, err := client.ExportInstance(ctx, &argocdv1.ExportInstanceRequest{
		OrganizationId: orgID,
		IdType:         idv1.Type_NAME,
		Id:             instance.ArgoCD.Name.ValueString(),
	})
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Argo CD instance. %s", err))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("---------export:%+v", exportResp))
	var argoCD *v1alpha1.ArgoCD
	err = marshal.RemarshalTo(exportResp.Argocd.AsMap(), &argoCD)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Argo CD instance. %s", err))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("---------export:%+v", argoCD))

	planClusters := map[string]types.Cluster{}
	for _, c := range instance.Clusters {
		planClusters[c.Name.ValueString()] = c
	}
	var clusters []types.Cluster
	for _, cluster := range exportResp.Clusters {
		var c *v1alpha1.Cluster
		err = marshal.RemarshalTo(cluster.AsMap(), &c)
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Argo CD instance. %s", err))
			return
		}
		cl := types.ToClusterTFModel(ctx, diagnostics, c)
		cl.Manifests = getManifests(ctx, diagnostics, client, orgID, instance.ID.ValueString(), cl.Name.ValueString())
		cl.Kubeconfig = planClusters[cl.Name.ValueString()].Kubeconfig
		clusters = append(clusters, cl)
	}
	instance.Clusters = clusters
	instance.ArgoCD = types.ToArgoCDTFModel(ctx, diagnostics, argoCD)
	instance.ArgoCDConfigMap.Update(ctx, diagnostics, exportResp.ArgocdConfigmap)
	instance.ArgoCDRBACConfigMap.Update(ctx, diagnostics, exportResp.ArgocdRbacConfigmap)
	instance.NotificationsConfigMap.Update(ctx, diagnostics, exportResp.NotificationsConfigmap)
	instance.ImageUpdaterConfigMap.Update(ctx, diagnostics, exportResp.ImageUpdaterConfigmap)
	instance.ImageUpdaterSSHConfigMap.Update(ctx, diagnostics, exportResp.ImageUpdaterSshConfigmap)
	//instance.ArgoCDKnownHostsConfigMap.Update(ctx, diagnostics, exportResp.ArgocdKnownHostsConfigmap)
	instance.ArgoCDTLSCertsConfigMap.Update(ctx, diagnostics, exportResp.ArgocdTlsCertsConfigmap)
	tflog.Info(ctx, fmt.Sprintf("---------instance:%+v", instance))
}

func getManifests(ctx context.Context, diagnostics *diag.Diagnostics, client argocdv1.ArgoCDServiceGatewayClient, orgId, instanceId, clusterName string) tftypes.String {
	clusterReq := &argocdv1.GetInstanceClusterRequest{
		OrganizationId: orgId,
		InstanceId:     instanceId,
		Id:             clusterName,
		IdType:         idv1.Type_NAME,
	}
	clusterResp, err := client.GetInstanceCluster(ctx, clusterReq)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance clusters, got error: %s", err))
		return tftypes.StringNull()
	}
	cluster, err := waitClusterReconStatus(ctx, client, clusterResp.GetCluster(), orgId, instanceId)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to check cluster reconciliation status. %s", err))
		return tftypes.StringNull()
	}
	apiReq := &argocdv1.GetInstanceClusterManifestsRequest{
		OrganizationId: orgId,
		InstanceId:     instanceId,
		Id:             cluster.Id,
	}
	apiResp, err := client.GetInstanceClusterManifests(ctx, apiReq)
	if err != nil {
		diagnostics.AddError("Akuity API error", fmt.Sprintf("Unable to download manifests: %s", err))
		return tftypes.StringNull()
	}
	return tftypes.StringValue(string(apiResp.GetData()))
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

func waitClusterHealthStatus(ctx context.Context, client argocdv1.ArgoCDServiceGatewayClient, cluster *argocdv1.Cluster, orgId, instanceId string) (*argocdv1.Cluster, error) {
	healthStatus := cluster.GetHealthStatus()
	breakStatusesHealth := []healthv1.StatusCode{healthv1.StatusCode_STATUS_CODE_HEALTHY, healthv1.StatusCode_STATUS_CODE_DEGRADED}

	for !slices.Contains(breakStatusesHealth, healthStatus.GetCode()) {
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
		healthStatus = cluster.GetHealthStatus()
		tflog.Debug(ctx, fmt.Sprintf("Cluster health status: %s", healthStatus.String()))
	}
	return cluster, nil
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

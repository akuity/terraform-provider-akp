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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	reconv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/reconciliation/v1"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
	"github.com/akuity/terraform-provider-akp/akp/conversion"
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
	var plan *types.Instance

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.MaskLogStrings(ctx, plan.GetSensitiveStrings()...)
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	apiReq := r.buildApplyRequest(ctx, resp.Diagnostics, plan)
	tflog.Debug(ctx, fmt.Sprintf("Api Req: %s", apiReq.String()))
	apiResp, err := r.akpCli.Cli.ApplyInstance(ctx, apiReq)
	tflog.Debug(ctx, fmt.Sprintf("Api Resp: %s", apiResp.String()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
		return
	}
	getInstanceReq := &argocdv1.GetInstanceRequest{
		OrganizationId: r.akpCli.OrgId,
		IdType:         idv1.Type_NAME,
		Id:             apiReq.Id,
	}

	tflog.Debug(ctx, fmt.Sprintf("API Req: %s", apiReq))
	getInstanceResp, err := r.akpCli.Cli.GetInstance(ctx, getInstanceReq)
	tflog.Debug(ctx, fmt.Sprintf("API Resp: %s", getInstanceResp))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Argo CD instance: %s", err))
		return
	}
	plan.ID = tftypes.StringValue(getInstanceResp.Instance.Id)
	r.refresh(ctx, resp.Diagnostics, plan)
	for _, cluster := range plan.Clusters {
		kcfg, diag := r.getKcfg(ctx, cluster.KubeConfig)
		resp.Diagnostics.Append(diag...)
		if resp.Diagnostics.HasError() {
			return
		}
		// Apply the manifests
		if kcfg != nil {
			clusterReq := &argocdv1.GetInstanceClusterRequest{
				OrganizationId: r.akpCli.OrgId,
				InstanceId:     plan.ID.ValueString(),
				Id:             cluster.Name.ValueString(),
				IdType:         idv1.Type_NAME,
			}
			tflog.Debug(ctx, fmt.Sprintf("Api Request: %s", clusterReq))
			clusterResp, err := r.akpCli.Cli.GetInstanceCluster(ctx, clusterReq)
			tflog.Debug(ctx, fmt.Sprintf("Api Response: %s", clusterResp))
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance clusters, got error: %s", err))
			}
			tflog.Info(ctx, "Applying the manifests...")
			tflog.Debug(ctx, fmt.Sprintf("-----Agent:%s", cluster.Manifests.ValueString()))
			resp.Diagnostics.Append(r.applyManifests(ctx, cluster.Manifests.ValueString(), kcfg)...)
			if resp.Diagnostics.HasError() {
				return
			}
			clusterResp.Cluster, err = r.waitClusterHealthStatus(ctx, clusterResp.Cluster, plan.ID.ValueString())
		}
	}
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

	r.refresh(ctx, resp.Diagnostics, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AkpInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Updating an AKP ArgoCD Instance")
	var plan *types.Instance

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.MaskLogStrings(ctx, plan.GetSensitiveStrings()...)
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	apiReq := r.buildApplyRequest(ctx, resp.Diagnostics, plan)
	tflog.Debug(ctx, fmt.Sprintf("Api Req: %s", apiReq.String()))
	apiResp, err := r.akpCli.Cli.ApplyInstance(ctx, apiReq)
	tflog.Debug(ctx, fmt.Sprintf("Api Resp: %s", apiResp.String()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Argo CD instance. %s", err))
		return
	}
	getInstanceReq := &argocdv1.GetInstanceRequest{
		OrganizationId: r.akpCli.OrgId,
		IdType:         idv1.Type_NAME,
		Id:             apiReq.Id,
	}

	tflog.Debug(ctx, fmt.Sprintf("API Req: %s", apiReq))
	getInstanceResp, err := r.akpCli.Cli.GetInstance(ctx, getInstanceReq)
	tflog.Debug(ctx, fmt.Sprintf("API Resp: %s", getInstanceResp))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Argo CD instance: %s", err))
		return
	}
	plan.ID = tftypes.StringValue(getInstanceResp.Instance.Id)
	r.refresh(ctx, resp.Diagnostics, plan)
	for _, cluster := range plan.Clusters {
		kcfg, diag := r.getKcfg(ctx, cluster.KubeConfig)
		resp.Diagnostics.Append(diag...)
		if resp.Diagnostics.HasError() {
			return
		}
		// Apply the manifests
		if kcfg != nil {
			clusterReq := &argocdv1.GetInstanceClusterRequest{
				OrganizationId: r.akpCli.OrgId,
				InstanceId:     plan.ID.ValueString(),
				Id:             cluster.Name.ValueString(),
				IdType:         idv1.Type_NAME,
			}
			tflog.Debug(ctx, fmt.Sprintf("Api Request: %s", clusterReq))
			clusterResp, err := r.akpCli.Cli.GetInstanceCluster(ctx, clusterReq)
			tflog.Debug(ctx, fmt.Sprintf("Api Response: %s", clusterResp))
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance clusters, got error: %s", err))
			}
			tflog.Info(ctx, "Applying the manifests...")
			tflog.Debug(ctx, fmt.Sprintf("-----Agent:%s", cluster.Manifests.ValueString()))
			resp.Diagnostics.Append(r.applyManifests(ctx, cluster.Manifests.ValueString(), kcfg)...)
			if resp.Diagnostics.HasError() {
				return
			}
			clusterResp.Cluster, err = r.waitClusterHealthStatus(ctx, clusterResp.Cluster, plan.ID.ValueString())
		}
	}
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

func (r *AkpInstanceResource) buildApplyRequest(ctx context.Context, diag diag.Diagnostics, instance *types.Instance) *argocdv1.ApplyInstanceRequest {
	applyReq := &argocdv1.ApplyInstanceRequest{
		OrganizationId:                r.akpCli.OrgId,
		IdType:                        idv1.Type_NAME,
		Id:                            instance.ArgoCD.Name.ValueString(),
		Argocd:                        buildArgoCD(ctx, diag, instance),
		ArgocdConfigmap:               buildConfigMap(ctx, diag, &instance.ArgoCDConfigMap, "argocd-cm"),
		ArgocdRbacConfigmap:           buildConfigMap(ctx, diag, &instance.ArgoCDRBACConfigMap, "argocd-rbac-cm"),
		ArgocdSecret:                  buildSecret(ctx, diag, &instance.ArgoCDSecret, "argocd-secret"),
		NotificationsConfigmap:        buildConfigMap(ctx, diag, &instance.NotificationsConfigMap, "argocd-notifications-cm"),
		NotificationsSecret:           buildSecret(ctx, diag, &instance.NotificationsSecret, "argocd-notifications-secret"),
		ImageUpdaterConfigmap:         buildConfigMap(ctx, diag, &instance.ImageUpdaterConfigMap, "argocd-image-updater-config"),
		ImageUpdaterSshConfigmap:      buildConfigMap(ctx, diag, &instance.ImageUpdaterSSHConfigMap, "argocd-image-updater-ssh-config"),
		ImageUpdaterSecret:            buildSecret(ctx, diag, &instance.ImageUpdaterSecret, "argocd-image-updater-secret"),
		Clusters:                      buildClusters(ctx, diag, instance),
		ArgocdKnownHostsConfigmap:     buildConfigMap(ctx, diag, &instance.ArgoCDKnownHostsConfigMap, "argocd-ssh-known-hosts-cm"),
		ArgocdTlsCertsConfigmap:       buildConfigMap(ctx, diag, &instance.ArgoCDTLSCertsConfigMap, "argocd-tls-certs-cm"),
		RepoCredentialSecrets:         buildSecrets(ctx, diag, instance.RepoCredentialSecrets),
		RepoTemplateCredentialSecrets: buildSecrets(ctx, diag, instance.RepoTemplateCredentialSecrets),
		PruneRepoCredentialSecrets:    true,
		PruneClusters:                 true,
	}
	return applyReq
}

func buildArgoCD(ctx context.Context, diag diag.Diagnostics, instance *types.Instance) *structpb.Struct {
	apiArgoCD := conversion.ToArgoCDAPIModel(ctx, diag, &instance.ArgoCD)
	argocdMap := map[string]interface{}{}
	if err := conversion.RemarshalTo(apiArgoCD, &argocdMap); err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to creat Argo CD instance. %s", err))
	}
	s, err := structpb.NewStruct(argocdMap)
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
		return nil
	}
	return s
}

func buildClusters(ctx context.Context, diag diag.Diagnostics, instance *types.Instance) []*structpb.Struct {
	var cs []*structpb.Struct
	tflog.Debug(ctx, fmt.Sprintf("%+v", instance.Clusters))
	for _, cluster := range instance.Clusters {
		apiCluster := conversion.ToClusterAPIModel(ctx, diag, &cluster)
		tflog.Debug(ctx, fmt.Sprintf("api%+v", apiCluster))
		m := map[string]interface{}{}
		if err := conversion.RemarshalTo(apiCluster, &m); err != nil {
			diag.AddError("Client Error", fmt.Sprintf("Unable to creat Argo CD instance. %s", err))
		}
		tflog.Debug(ctx, fmt.Sprintf("m%+v", m))
		s, err := structpb.NewStruct(m)
		if err != nil {
			diag.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
			return nil
		}
		tflog.Debug(ctx, fmt.Sprintf("s%+v", s))
		cs = append(cs, s)
	}
	tflog.Debug(ctx, fmt.Sprintf("%+v", cs))
	return cs
}

func buildSecrets(ctx context.Context, diag diag.Diagnostics, secrets []types.Secret) []*structpb.Struct {
	var res []*structpb.Struct
	for _, secret := range secrets {
		res = append(res, buildSecret(ctx, diag, &secret, secret.Name.ValueString()))
	}
	return res
}

func buildConfigMap(ctx context.Context, diag diag.Diagnostics, cm *types.ConfigMap, name string) *structpb.Struct {
	apiModel := conversion.ToConfigMapAPIModel(ctx, diag, cm, name)
	m := map[string]interface{}{}
	if err := conversion.RemarshalTo(apiModel, &m); err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to creat Argo CD instance. %s", err))
	}
	configMap, err := structpb.NewStruct(m)
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
		return nil
	}
	return configMap
}

func buildSecret(ctx context.Context, diag diag.Diagnostics, secret *types.Secret, name string) *structpb.Struct {
	apiModel := conversion.ToSecretAPIModel(ctx, diag, secret, name)
	m := map[string]interface{}{}
	if err := conversion.RemarshalTo(apiModel, &m); err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to creat Argo CD instance. %s", err))
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
		return nil
	}
	return s
}

func (r *AkpInstanceResource) refresh(ctx context.Context, diagnostics diag.Diagnostics, x *types.Instance) {

	exportResp, err := r.akpCli.Cli.ExportInstance(ctx, &argocdv1.ExportInstanceRequest{
		OrganizationId: r.akpCli.OrgId,
		IdType:         idv1.Type_NAME,
		Id:             x.ArgoCD.Name.ValueString(),
	})
	tflog.Debug(ctx, fmt.Sprintf("Export Resp: %s", exportResp.String()))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Argo CD instance. %s", err))
		return
	}
	var argoCD *v1alpha1.ArgoCD
	err = conversion.RemarshalTo(exportResp.Argocd.AsMap(), &argoCD)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Argo CD instance. %s", err))
		return
	}

	planClusters := map[string]types.Cluster{}
	for _, c := range x.Clusters {
		planClusters[c.Name.ValueString()] = c
	}
	var clusters []types.Cluster
	for _, cluster := range exportResp.Clusters {
		var c *v1alpha1.Cluster
		err = conversion.RemarshalTo(cluster.AsMap(), &c)
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Argo CD instance. %s", err))
			return
		}
		cl := *conversion.ToClusterTFModel(ctx, diagnostics, c)
		cl.Manifests = r.getManifests(ctx, diagnostics, r.akpCli.OrgId, x.ID.ValueString(), cl.Name.ValueString())
		cl.KubeConfig = planClusters[cl.Name.ValueString()].KubeConfig

		clusters = append(clusters, cl)
	}

	x.ArgoCD = *conversion.ToArgoCDTFModel(ctx, diagnostics, argoCD)
	x.Clusters = clusters

	x.ArgoCDConfigMap = buildTFConfigMap(ctx, diagnostics, x.ArgoCDConfigMap, exportResp.ArgocdConfigmap, "argocd-cm")
	x.ArgoCDRBACConfigMap = buildTFConfigMap(ctx, diagnostics, x.ArgoCDRBACConfigMap, exportResp.ArgocdRbacConfigmap, "argocd-rbac-cm")
	x.NotificationsConfigMap = buildTFConfigMap(ctx, diagnostics, x.NotificationsConfigMap, exportResp.NotificationsConfigmap, "argocd-notifications-cm")
	x.ImageUpdaterConfigMap = buildTFConfigMap(ctx, diagnostics, x.ImageUpdaterConfigMap, exportResp.ImageUpdaterConfigmap, "argocd-image-updater-config")
	x.ImageUpdaterSSHConfigMap = buildTFConfigMap(ctx, diagnostics, x.ImageUpdaterSSHConfigMap, exportResp.ImageUpdaterSshConfigmap, "argocd-image-updater-ssh-config")
	x.ArgoCDTLSCertsConfigMap = buildTFConfigMap(ctx, diagnostics, x.ArgoCDTLSCertsConfigMap, exportResp.ArgocdTlsCertsConfigmap, "argocd-tls-certs-cm")
	tflog.Debug(ctx, fmt.Sprintf("refreash---------%+v", x))

}

func buildTFConfigMap(ctx context.Context, diagnostics diag.Diagnostics, cm types.ConfigMap, data *structpb.Struct, name string) types.ConfigMap {
	m := map[string]string{}
	err := conversion.RemarshalTo(data, &m)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Argo CD instance. %s", err))
		return types.ConfigMap{}
	}
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: commonLabels(name)},
		Data:       m,
	}
	newCM := conversion.ToConfigMapTFModel(ctx, diagnostics, configMap)
	cm.Data = mergeStringMaps(ctx, diagnostics, cm.Data, newCM.Data)
	return cm
}

func commonLabels(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":    name,
		"app.kubernetes.io/part-of": "argocd",
	}
}

func mergeStringMaps(ctx context.Context, diagnostics diag.Diagnostics, old, new tftypes.Map) tftypes.Map {
	var oldData, newData map[string]string
	if !new.IsNull() {
		diagnostics.Append(new.ElementsAs(ctx, &newData, true)...)
	} else {
		newData = make(map[string]string)
	}
	if !old.IsNull() {
		diagnostics.Append(old.ElementsAs(ctx, &oldData, true)...)
	} else {
		oldData = make(map[string]string)
	}
	res := make(map[string]string)
	for name := range oldData {
		if val, ok := newData[name]; ok {
			res[name] = val
		} else {
			delete(res, name)
		}
	}
	resMap, d := tftypes.MapValueFrom(ctx, tftypes.StringType, res)
	diagnostics.Append(d...)
	return resMap
}

func (r *AkpInstanceResource) getManifests(ctx context.Context, diags diag.Diagnostics, orgId, instanceId, Id string) tftypes.String {
	clusterReq := &argocdv1.GetInstanceClusterRequest{
		OrganizationId: orgId,
		InstanceId:     instanceId,
		Id:             Id,
		IdType:         idv1.Type_NAME,
	}
	tflog.Debug(ctx, fmt.Sprintf("Api Request: %s", clusterReq))
	clusterResp, err := r.akpCli.Cli.GetInstanceCluster(ctx, clusterReq)
	tflog.Debug(ctx, fmt.Sprintf("Api Response: %s", clusterResp))
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read instance clusters, got error: %s", err))
		return tftypes.StringNull()
	}
	cluster, err := r.waitClusterReconStatus(ctx, clusterResp.GetCluster(), instanceId)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to check cluster reconciliation status. %s", err))
		return tftypes.StringNull()
	}
	apiReq := &argocdv1.GetInstanceClusterManifestsRequest{
		OrganizationId: orgId,
		InstanceId:     instanceId,
		Id:             cluster.Id,
	}
	tflog.Debug(ctx, fmt.Sprintf("------manifest apiReq: %s", apiReq))
	apiResp, err := r.akpCli.Cli.GetInstanceClusterManifests(ctx, apiReq)
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("------manifest apiReq err: %s", err.Error()))
		diags.AddError("Akuity API error", fmt.Sprintf("Unable to download manifests: %s", err))
		return tftypes.StringNull()
	}
	tflog.Debug(ctx, fmt.Sprintf("-------manifest apiResp: %s", apiResp))
	return tftypes.StringValue(string(apiResp.GetData()))
}

func (r *AkpInstanceResource) getKcfg(ctx context.Context, kubeConf tftypes.Object) (*rest.Config, diag.Diagnostics) {
	var kubeConfig kube.KubeConfig
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

func (r *AkpInstanceResource) applyManifests(ctx context.Context, manifests string, cfg *rest.Config) diag.Diagnostics {
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

func (r *AkpInstanceResource) deleteManifests(ctx context.Context, manifests string, cfg *rest.Config) diag.Diagnostics {
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

func (r *AkpInstanceResource) waitClusterHealthStatus(ctx context.Context, cluster *argocdv1.Cluster, instanceId string) (*argocdv1.Cluster, error) {
	healthStatus := cluster.GetHealthStatus()
	breakStatusesHealth := []healthv1.StatusCode{healthv1.StatusCode_STATUS_CODE_HEALTHY, healthv1.StatusCode_STATUS_CODE_DEGRADED}

	for !slices.Contains(breakStatusesHealth, healthStatus.GetCode()) {
		time.Sleep(1 * time.Second)
		apiResp, err := r.akpCli.Cli.GetInstanceCluster(ctx, &argocdv1.GetInstanceClusterRequest{
			OrganizationId: r.akpCli.OrgId,
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

func (r *AkpInstanceResource) waitClusterReconStatus(ctx context.Context, cluster *argocdv1.Cluster, instanceId string) (*argocdv1.Cluster, error) {
	reconStatus := cluster.GetReconciliationStatus()
	breakStatusesRecon := []reconv1.StatusCode{reconv1.StatusCode_STATUS_CODE_SUCCESSFUL, reconv1.StatusCode_STATUS_CODE_FAILED}

	for !slices.Contains(breakStatusesRecon, reconStatus.GetCode()) {
		time.Sleep(1 * time.Second)
		apiResp, err := r.akpCli.Cli.GetInstanceCluster(ctx, &argocdv1.GetInstanceClusterRequest{
			OrganizationId: r.akpCli.OrgId,
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

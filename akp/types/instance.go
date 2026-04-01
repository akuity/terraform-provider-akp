package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/structpb"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
)

type Instance struct {
	ID                            types.String                       `tfsdk:"id"`
	Name                          types.String                       `tfsdk:"name"`
	Workspace                     types.String                       `tfsdk:"workspace"`
	ArgoCD                        *ArgoCD                            `tfsdk:"argocd"`
	ArgoCDConfigMap               types.Map                          `tfsdk:"argocd_cm"`
	ArgoCDRBACConfigMap           types.Map                          `tfsdk:"argocd_rbac_cm"`
	ArgoCDSecret                  types.Map                          `tfsdk:"argocd_secret"`
	ApplicationSetSecret          types.Map                          `tfsdk:"application_set_secret"`
	NotificationsConfigMap        types.Map                          `tfsdk:"argocd_notifications_cm"`
	NotificationsSecret           types.Map                          `tfsdk:"argocd_notifications_secret"`
	ImageUpdaterConfigMap         types.Map                          `tfsdk:"argocd_image_updater_config"`
	ImageUpdaterSSHConfigMap      types.Map                          `tfsdk:"argocd_image_updater_ssh_config"`
	ImageUpdaterSecret            types.Map                          `tfsdk:"argocd_image_updater_secret"`
	ArgoCDKnownHostsConfigMap     types.Map                          `tfsdk:"argocd_ssh_known_hosts_cm"`
	ArgoCDTLSCertsConfigMap       types.Map                          `tfsdk:"argocd_tls_certs_cm"`
	RepoCredentialSecrets         types.Map                          `tfsdk:"repo_credential_secrets"`
	RepoTemplateCredentialSecrets types.Map                          `tfsdk:"repo_template_credential_secrets"`
	ConfigManagementPlugins       map[string]*ConfigManagementPlugin `tfsdk:"config_management_plugins"`
	ArgoCDResources               types.Map                          `tfsdk:"argocd_resources"`
}

func (i *Instance) GetSensitiveStrings(ctx context.Context, diagnostics *diag.Diagnostics) []string {
	var res []string
	res = append(res, GetSensitiveStrings(i.ArgoCDSecret)...)
	res = append(res, GetSensitiveStrings(i.NotificationsSecret)...)
	res = append(res, GetSensitiveStrings(i.ImageUpdaterSecret)...)
	res = append(res, GetSensitiveStrings(i.ApplicationSetSecret)...)
	var repoCredentialSecrets map[string]types.Map
	if !i.RepoCredentialSecrets.IsNull() {
		diagnostics.Append(i.RepoCredentialSecrets.ElementsAs(ctx, &repoCredentialSecrets, true)...)
	}
	for _, secret := range repoCredentialSecrets {
		res = append(res, GetSensitiveStrings(secret)...)
	}
	var repoTemplateCredentialSecrets map[string]types.Map
	if !i.RepoTemplateCredentialSecrets.IsNull() {
		diagnostics.Append(i.RepoTemplateCredentialSecrets.ElementsAs(ctx, &repoTemplateCredentialSecrets, true)...)
	}
	for _, secret := range repoTemplateCredentialSecrets {
		res = append(res, GetSensitiveStrings(secret)...)
	}
	return res
}

func (i *Instance) Update(ctx context.Context, diagnostics *diag.Diagnostics, exportResp *argocdv1.ExportInstanceResponse, isDataSource bool) error {
	if i.ArgoCD == nil {
		i.ArgoCD = &ArgoCD{}
	}
	apiMap := exportResp.Argocd.AsMap()
	if isDataSource {
		diagnostics.Append(BuildStateFromAPI(ctx, apiMap, i.ArgoCD, nil, ReverseOverridesMap, ReverseRenamesMap, "argocd")...)
	} else {
		plan := DeepCopyArgoCD(i.ArgoCD)
		diagnostics.Append(BuildStateFromAPI(ctx, apiMap, i.ArgoCD, plan, ReverseOverridesMap, ReverseRenamesMap, "argocd")...)
	}
	if isDataSource {
		i.ArgoCDConfigMap = ToDataSourceConfigMapTFModel(ctx, diagnostics, exportResp.ArgocdConfigmap, i.ArgoCDConfigMap)
	} else {
		i.ArgoCDConfigMap = ToConfigMapTFModel(ctx, diagnostics, exportResp.ArgocdConfigmap, i.ArgoCDConfigMap)
	}
	i.ArgoCDRBACConfigMap = ToConfigMapTFModel(ctx, diagnostics, exportResp.ArgocdRbacConfigmap, i.ArgoCDRBACConfigMap)
	i.NotificationsConfigMap = ToConfigMapTFModel(ctx, diagnostics, exportResp.NotificationsConfigmap, i.NotificationsConfigMap)
	i.ImageUpdaterConfigMap = ToConfigMapTFModel(ctx, diagnostics, exportResp.ImageUpdaterConfigmap, i.ImageUpdaterConfigMap)
	i.ImageUpdaterSSHConfigMap = ToConfigMapTFModel(ctx, diagnostics, exportResp.ImageUpdaterSshConfigmap, i.ImageUpdaterSSHConfigMap)
	i.ArgoCDTLSCertsConfigMap = ToConfigMapTFModel(ctx, diagnostics, exportResp.ArgocdTlsCertsConfigmap, i.ArgoCDTLSCertsConfigMap)
	i.ArgoCDKnownHostsConfigMap = ToConfigMapTFModel(ctx, diagnostics, exportResp.ArgocdKnownHostsConfigmap, i.ArgoCDKnownHostsConfigMap)
	i.ConfigManagementPlugins = ToConfigManagementPluginsTFModel(ctx, diagnostics, exportResp.ConfigManagementPlugins, i.ConfigManagementPlugins)
	if err := i.syncArgoResources(ctx, exportResp, diagnostics, isDataSource); err != nil {
		return err
	}
	return nil
}

func (i *Instance) syncArgoResources(
	ctx context.Context,
	exportResp *argocdv1.ExportInstanceResponse,
	diagnostics *diag.Diagnostics,
	isDataSource bool,
) error {
	appliedResources := make([]*structpb.Struct, 0)
	appliedResources = append(appliedResources, exportResp.Applications...)
	appliedResources = append(appliedResources, exportResp.ApplicationSets...)
	appliedResources = append(appliedResources, exportResp.AppProjects...)

	newMap, err := syncResources(
		ctx,
		diagnostics,
		i.ArgoCDResources,
		appliedResources,
		"ArgoCD",
		isDataSource,
	)
	if err != nil {
		return err
	}
	i.ArgoCDResources = newMap
	return nil
}

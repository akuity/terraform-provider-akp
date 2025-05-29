package types

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
)

type Instance struct {
	ID                            types.String                       `tfsdk:"id"`
	Name                          types.String                       `tfsdk:"name"`
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

func (i *Instance) Update(ctx context.Context, diagnostics *diag.Diagnostics, exportResp *argocdv1.ExportInstanceResponse) error {
	var argoCD *v1alpha1.ArgoCD
	err := marshal.RemarshalTo(exportResp.Argocd.AsMap(), &argoCD)
	if err != nil {
		return errors.Wrap(err, "Unable to get Argo CD instance")
	}
	if i.ArgoCD == nil {
		i.ArgoCD = &ArgoCD{}
	}
	if argoCD.Spec.InstanceSpec.Fqdn == nil {
		fqdn := ""
		argoCD.Spec.InstanceSpec.Fqdn = &fqdn
	}
	i.ArgoCD.Update(ctx, diagnostics, argoCD)
	i.ArgoCDConfigMap = ToFilteredConfigMapTFModel(ctx, diagnostics, exportResp.ArgocdConfigmap, i.ArgoCDConfigMap)
	i.ArgoCDRBACConfigMap = ToConfigMapTFModel(ctx, diagnostics, exportResp.ArgocdRbacConfigmap, i.ArgoCDRBACConfigMap)
	i.NotificationsConfigMap = ToConfigMapTFModel(ctx, diagnostics, exportResp.NotificationsConfigmap, i.NotificationsConfigMap)
	i.ImageUpdaterConfigMap = ToConfigMapTFModel(ctx, diagnostics, exportResp.ImageUpdaterConfigmap, i.ImageUpdaterConfigMap)
	i.ImageUpdaterSSHConfigMap = ToConfigMapTFModel(ctx, diagnostics, exportResp.ImageUpdaterSshConfigmap, i.ImageUpdaterSSHConfigMap)
	i.ArgoCDTLSCertsConfigMap = ToConfigMapTFModel(ctx, diagnostics, exportResp.ArgocdTlsCertsConfigmap, i.ArgoCDTLSCertsConfigMap)
	i.ArgoCDKnownHostsConfigMap = ToConfigMapTFModel(ctx, diagnostics, exportResp.ArgocdKnownHostsConfigmap, i.ArgoCDKnownHostsConfigMap)
	i.ConfigManagementPlugins = ToConfigManagementPluginsTFModel(ctx, diagnostics, exportResp.ConfigManagementPlugins, i.ConfigManagementPlugins)
	if err := i.syncArgoResources(ctx, exportResp, diagnostics); err != nil {
		return err
	}
	return nil
}

func (i *Instance) syncArgoResources(
	ctx context.Context,
	exportResp *argocdv1.ExportInstanceResponse,
	diagnostics *diag.Diagnostics,
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
	)
	if err != nil {
		return err
	}
	i.ArgoCDResources = newMap
	return nil
}

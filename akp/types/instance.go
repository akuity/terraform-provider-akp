package types

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
)

type Instance struct {
	ID                            types.String `tfsdk:"id"`
	Name                          types.String `tfsdk:"name"`
	ArgoCD                        *ArgoCD      `tfsdk:"argocd"`
	ArgoCDConfigMap               *ConfigMap   `tfsdk:"argocd_cm"`
	ArgoCDRBACConfigMap           *ConfigMap   `tfsdk:"argocd_rbac_cm"`
	ArgoCDSecret                  *Secret      `tfsdk:"argocd_secret"`
	NotificationsConfigMap        *ConfigMap   `tfsdk:"argocd_notifications_cm"`
	NotificationsSecret           *Secret      `tfsdk:"argocd_notifications_secret"`
	ImageUpdaterConfigMap         *ConfigMap   `tfsdk:"argocd_image_updater_config"`
	ImageUpdaterSSHConfigMap      *ConfigMap   `tfsdk:"argocd_image_updater_ssh_config"`
	ImageUpdaterSecret            *Secret      `tfsdk:"argocd_image_updater_secret"`
	ArgoCDKnownHostsConfigMap     *ConfigMap   `tfsdk:"argocd_ssh_known_hosts_cm"`
	ArgoCDTLSCertsConfigMap       *ConfigMap   `tfsdk:"argocd_tls_certs_cm"`
	RepoCredentialSecrets         []Secret     `tfsdk:"repo_credential_secrets"`
	RepoTemplateCredentialSecrets []Secret     `tfsdk:"repo_template_credential_secrets"`
}

func (i *Instance) GetSensitiveStrings() []string {
	var res []string
	res = append(res, i.ArgoCDSecret.GetSensitiveStrings()...)
	res = append(res, i.NotificationsSecret.GetSensitiveStrings()...)
	res = append(res, i.ImageUpdaterSecret.GetSensitiveStrings()...)
	for _, secret := range i.RepoCredentialSecrets {
		res = append(res, secret.GetSensitiveStrings()...)
	}
	for _, secret := range i.RepoTemplateCredentialSecrets {
		res = append(res, secret.GetSensitiveStrings()...)
	}
	return res
}

func (i *Instance) Update(ctx context.Context, diagnostics *diag.Diagnostics, exportResp *argocdv1.ExportInstanceResponse) {
	var argoCD *v1alpha1.ArgoCD
	err := marshal.RemarshalTo(exportResp.Argocd.AsMap(), &argoCD)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Argo CD instance. %s", err))
		return
	}
	if i.ArgoCD == nil {
		i.ArgoCD = &ArgoCD{}
	}
	i.ArgoCD.Update(ctx, diagnostics, argoCD)
	if i.ArgoCDConfigMap == nil {
		i.ArgoCDConfigMap = &ConfigMap{}
	}
	i.ArgoCDConfigMap.Update(ctx, diagnostics, exportResp.ArgocdConfigmap)
	if i.ArgoCDRBACConfigMap == nil {
		i.ArgoCDRBACConfigMap = &ConfigMap{}
	}
	i.ArgoCDRBACConfigMap.Update(ctx, diagnostics, exportResp.ArgocdRbacConfigmap)
	if i.NotificationsConfigMap == nil {
		i.NotificationsConfigMap = &ConfigMap{}
	}
	i.NotificationsConfigMap.Update(ctx, diagnostics, exportResp.NotificationsConfigmap)
	if i.ImageUpdaterConfigMap == nil {
		i.ImageUpdaterConfigMap = &ConfigMap{}
	}
	i.ImageUpdaterConfigMap.Update(ctx, diagnostics, exportResp.ImageUpdaterConfigmap)
	if i.ImageUpdaterSSHConfigMap == nil {
		i.ImageUpdaterSSHConfigMap = &ConfigMap{}
	}
	i.ImageUpdaterSSHConfigMap.Update(ctx, diagnostics, exportResp.ImageUpdaterSshConfigmap)
	if i.ArgoCDTLSCertsConfigMap == nil {
		i.ArgoCDTLSCertsConfigMap = &ConfigMap{}
	}
	i.ArgoCDTLSCertsConfigMap.Update(ctx, diagnostics, exportResp.ArgocdTlsCertsConfigmap)
	if i.ArgoCDKnownHostsConfigMap == nil {
		i.ArgoCDKnownHostsConfigMap = &ConfigMap{}
	}
	i.ArgoCDKnownHostsConfigMap.Update(ctx, diagnostics, exportResp.ArgocdKnownHostsConfigmap)
}

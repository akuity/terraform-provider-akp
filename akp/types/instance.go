package types

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Instance struct {
	ID                            types.String `json:"id,omitempty" tfsdk:"id"`
	Name                          types.String `json:"name,omitempty" tfsdk:"name"`
	ArgoCD                        *ArgoCD      `json:"argoCD,omitempty" tfsdk:"argocd"`
	ArgoCDConfigMap               *ConfigMap   `json:"argoCDConfigMap,omitempty" tfsdk:"argocd_cm"`
	ArgoCDRBACConfigMap           *ConfigMap   `json:"argoCDRBACConfigMap,omitempty" tfsdk:"argocd_rbac_cm"`
	ArgoCDSecret                  *Secret      `json:"argoCDSecret,omitempty" tfsdk:"argocd_secret"`
	NotificationsConfigMap        *ConfigMap   `json:"notificationsConfigMap,omitempty" tfsdk:"argocd_notifications_cm"`
	NotificationsSecret           *Secret      `json:"notificationsSecret,omitempty" tfsdk:"argocd_notifications_secret"`
	ImageUpdaterConfigMap         *ConfigMap   `json:"imageUpdaterConfigMap,omitempty" tfsdk:"argocd_image_updater_config"`
	ImageUpdaterSSHConfigMap      *ConfigMap   `json:"imageUpdaterSSHConfigmap,omitempty" tfsdk:"argocd_image_updater_ssh_config"`
	ImageUpdaterSecret            *Secret      `json:"imageUpdaterSecret,omitempty" tfsdk:"argocd_image_updater_secret"`
	ArgoCDKnownHostsConfigMap     *ConfigMap   `json:"argoCDKnownHostsConfigMap,omitempty" tfsdk:"argocd_ssh_known_hosts_cm"`
	ArgoCDTLSCertsConfigMap       *ConfigMap   `json:"argoCDTLSCertsConfigMap,omitempty" tfsdk:"argocd_tls_certs_cm"`
	RepoCredentialSecrets         []Secret     `json:"repoCredentialSecrets,omitempty" tfsdk:"repo_credential_secrets"`
	RepoTemplateCredentialSecrets []Secret     `json:"repoTemplateCredentialSecrets,omitempty" tfsdk:"repo_template_credential_secrets"`
}

func (x *Instance) GetSensitiveStrings() []string {
	var res []string
	res = append(res, x.ArgoCDSecret.GetSensitiveStrings()...)
	res = append(res, x.NotificationsSecret.GetSensitiveStrings()...)
	res = append(res, x.ImageUpdaterSecret.GetSensitiveStrings()...)
	for _, secret := range x.RepoCredentialSecrets {
		res = append(res, secret.GetSensitiveStrings()...)
	}
	for _, secret := range x.RepoTemplateCredentialSecrets {
		res = append(res, secret.GetSensitiveStrings()...)
	}
	return res
}

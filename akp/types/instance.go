package types

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/structpb"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
)

type Instance struct {
	ID                            types.String `tfsdk:"id"`
	Name                          types.String `tfsdk:"name"`
	ArgoCD                        *ArgoCD      `tfsdk:"argocd"`
	ArgoCDConfigMap               types.Object `tfsdk:"argocd_cm"`
	ArgoCDRBACConfigMap           types.Object `tfsdk:"argocd_rbac_cm"`
	ArgoCDSecret                  *Secret      `tfsdk:"argocd_secret"`
	NotificationsConfigMap        types.Object `tfsdk:"argocd_notifications_cm"`
	NotificationsSecret           *Secret      `tfsdk:"argocd_notifications_secret"`
	ImageUpdaterConfigMap         types.Object `tfsdk:"argocd_image_updater_config"`
	ImageUpdaterSSHConfigMap      types.Object `tfsdk:"argocd_image_updater_ssh_config"`
	ImageUpdaterSecret            *Secret      `tfsdk:"argocd_image_updater_secret"`
	ArgoCDKnownHostsConfigMap     types.Object `tfsdk:"argocd_ssh_known_hosts_cm"`
	ArgoCDTLSCertsConfigMap       types.Object `tfsdk:"argocd_tls_certs_cm"`
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
	i.ArgoCDConfigMap = i.UpdateConfigMapObj(ctx, diagnostics, exportResp.ArgocdConfigmap)
	i.ArgoCDRBACConfigMap = i.UpdateConfigMapObj(ctx, diagnostics, exportResp.ArgocdRbacConfigmap)
	i.NotificationsConfigMap = i.UpdateConfigMapObj(ctx, diagnostics, exportResp.NotificationsConfigmap)
	i.ImageUpdaterConfigMap = i.UpdateConfigMapObj(ctx, diagnostics, exportResp.ImageUpdaterConfigmap)
	i.ImageUpdaterSSHConfigMap = i.UpdateConfigMapObj(ctx, diagnostics, exportResp.ImageUpdaterSshConfigmap)
	i.ArgoCDTLSCertsConfigMap = i.UpdateConfigMapObj(ctx, diagnostics, exportResp.ArgocdTlsCertsConfigmap)
	i.ArgoCDKnownHostsConfigMap = i.UpdateConfigMapObj(ctx, diagnostics, exportResp.ArgocdKnownHostsConfigmap)
}

func (i *Instance) UpdateConfigMapObj(ctx context.Context, diagnostics *diag.Diagnostics, data *structpb.Struct) types.Object {
	if data == nil || len(data.AsMap()) == 0 {
		return types.ObjectNull(configMapAttrTypes)
	}
	cm := &ConfigMap{}
	cm.Update(ctx, diagnostics, data)
	cmObject, diag := types.ObjectValueFrom(ctx, configMapAttrTypes, cm)
	diagnostics.Append(diag...)
	if diagnostics.HasError() {
		return types.ObjectNull(configMapAttrTypes)
	}
	return cmObject
}

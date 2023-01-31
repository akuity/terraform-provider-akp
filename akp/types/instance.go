package types

import (
	"context"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type AkpInstance struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Version       types.String `tfsdk:"version"`
	Description   types.String `tfsdk:"description"`
	Hostname      types.String `tfsdk:"hostname"`
	RbacConfig    types.Object `tfsdk:"rbac_config"`
	Config        types.Object `tfsdk:"config"`
}

func (x *AkpInstance) UpdateFrom(p *argocdv1.Instance) diag.Diagnostics {
	rbacConfigObject := &AkpArgoCDRBACConfig{}
	configObject := &AkpArgoCDConfig{}
	diags := diag.Diagnostics{}
	d := diag.Diagnostics{}
	x.Id = types.StringValue(p.Id)
	x.Name = types.StringValue(p.GetName())
	x.Version = types.StringValue(p.GetVersion())
	x.Description = types.StringValue(p.GetDescription())
	x.Hostname = types.StringValue(p.GetHostname())
	diags.Append(rbacConfigObject.UpdateObject(p.GetRbacConfig())...)
	x.RbacConfig, d = types.ObjectValueFrom(context.Background(), RBACConfigMapAttrTypes, rbacConfigObject)
	diags.Append(d...)
	diags.Append(configObject.UpdateObject(p.GetConfig())...)
	x.Config, d = types.ObjectValueFrom(context.Background(), configMapAttrTypes, configObject)
	diags.Append(d...)
	return diags
}

func (x *AkpInstance) As(target *argocdv1.Instance) diag.Diagnostics {
	rbacConfigTF := AkpArgoCDRBACConfig{}
	configTF := AkpArgoCDConfig{}
	diags := diag.Diagnostics{}
	target.Name = x.Name.ValueString()
	target.Description = x.Description.ValueString()
	target.Version = x.Version.ValueString()
	diags.Append(x.RbacConfig.As(context.Background(), &rbacConfigTF, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty: false,
		UnhandledUnknownAsEmpty: true,
	})...)
	diags.Append(rbacConfigTF.As(target.RbacConfig)...)
	diags.Append(x.Config.As(context.Background(), &configTF, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty: false,
		UnhandledUnknownAsEmpty: true,
	})...)
	diags.Append(configTF.As(target.Config)...)
	return diags
}

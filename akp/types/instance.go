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
	RbacConfigMap types.Object `tfsdk:"rbac_config"`
}

func (x *AkpInstance) UpdateFrom(p *argocdv1.Instance) diag.Diagnostics {
	rbacConfigMapObject := &AkpArgoCDRBACConfig{}
	diags := diag.Diagnostics{}
	d := diag.Diagnostics{}
	x.Id = types.StringValue(p.Id)
	x.Name = types.StringValue(p.GetName())
	x.Version = types.StringValue(p.GetVersion())
	x.Description = types.StringValue(p.GetDescription())
	x.Hostname = types.StringValue(p.GetHostname())
	diags.Append(rbacConfigMapObject.UpdateObject(p.RbacConfig)...)
	x.RbacConfigMap, d = types.ObjectValueFrom(context.Background(), RBACConfigMapAttrTypes, rbacConfigMapObject)
	diags.Append(d...)
	return diags
}

func (x *AkpInstance) As(target *argocdv1.Instance) diag.Diagnostics {
	rbacConfigTF := AkpArgoCDRBACConfig{}
	target.Name = x.Name.ValueString()
	target.Description = x.Description.ValueString()
	target.Version = x.Version.ValueString()
	diags := x.RbacConfigMap.As(context.Background(), &rbacConfigTF, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty: false,
		UnhandledUnknownAsEmpty: true,
	})
	rbacConfig := &argocdv1.ArgoCDRBACConfigMap{}
	diags.Append(rbacConfigTF.As(rbacConfig)...)
	target.RbacConfig = rbacConfig
	return diags
}

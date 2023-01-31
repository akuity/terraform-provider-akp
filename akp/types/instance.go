package types

import (
	"context"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type AkpInstance struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Version     types.String `tfsdk:"version"`
	Description types.String `tfsdk:"description"`
	Hostname    types.String `tfsdk:"hostname"`
	RbacConfig  types.Object `tfsdk:"rbac_config"`
	Config      types.Object `tfsdk:"config"`
	Spec        types.Object `tfsdk:"spec"`
}

func (x *AkpInstance) UpdateFrom(p *argocdv1.Instance) diag.Diagnostics {
	diags := diag.Diagnostics{}
	d := diag.Diagnostics{}
	x.Id = types.StringValue(p.Id)
	x.Name = types.StringValue(p.GetName())
	x.Version = types.StringValue(p.GetVersion())
	x.Description = types.StringValue(p.GetDescription())
	x.Hostname = types.StringValue(p.GetHostname())
	if p.RbacConfig == nil {
		x.RbacConfig = types.ObjectNull(RBACConfigMapAttrTypes)
	} else {
		rbacConfig := &AkpArgoCDRBACConfig{}
		diags.Append(rbacConfig.UpdateObject(p.RbacConfig)...)
		x.RbacConfig, d = types.ObjectValueFrom(context.Background(), RBACConfigMapAttrTypes, rbacConfig)
		diags.Append(d...)
	}

	if p.Config == nil {
		x.Config = types.ObjectNull(configMapAttrTypes)
	} else {
		config := &AkpArgoCDConfig{}
		diags.Append(config.UpdateObject(p.Config)...)
		x.Config, d = types.ObjectValueFrom(context.Background(), configMapAttrTypes, config)
		diags.Append(d...)
	}

	if p.Spec == nil {
		x.Spec = types.ObjectNull(instanceSpecAttrTypes)
	} else {
		spec := &AkpInstanceSpec{}
		diags.Append(spec.UpdateObject(p.Spec)...)
		x.Spec, d = types.ObjectValueFrom(context.Background(), instanceSpecAttrTypes, spec)
		diags.Append(d...)
	}

	return diags
}

func (x *AkpInstance) As(target *argocdv1.Instance) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Name = x.Name.ValueString()
	target.Description = x.Description.ValueString()
	target.Version = x.Version.ValueString()

	if x.RbacConfig.IsNull() {
		target.RbacConfig = nil
	} else if !x.RbacConfig.IsUnknown() {
		rbacConfig := AkpArgoCDRBACConfig{}
		targetRbacConfig := argocdv1.ArgoCDRBACConfigMap{}
		diags.Append(x.RbacConfig.As(context.Background(), &rbacConfig, basetypes.ObjectAsOptions{})...)
		diags.Append(rbacConfig.As(&targetRbacConfig)...)
		target.RbacConfig = &targetRbacConfig
	}

	if x.Config.IsNull() {
		target.Config = nil
	} else if !x.Config.IsUnknown() {
		config := AkpArgoCDConfig{}
		targetConfig := argocdv1.ArgoCDConfigMap{}
		diags.Append(x.Config.As(context.Background(), &config, basetypes.ObjectAsOptions{})...)
		diags.Append(config.As(&targetConfig)...)
		target.Config = &targetConfig
	}

	if x.Spec.IsNull() {
		target.Spec = nil
	} else if !x.Spec.IsUnknown() {
		spec := AkpInstanceSpec{}
		targetSpec := argocdv1.InstanceSpec{}
		diags.Append(x.Spec.As(context.Background(), &spec, basetypes.ObjectAsOptions{})...)
		diags.Append(spec.As(&targetSpec)...)
		target.Spec = &targetSpec
	}
	return diags
}

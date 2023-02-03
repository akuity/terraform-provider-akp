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

func MergeInstance(state *AkpInstance, plan *AkpInstance) (*AkpInstance, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpInstance{
		Id:       state.Id,
		Name:     state.Name,
		Version:  state.Version,
		Hostname: state.Hostname,
	}
	if plan.Description.IsUnknown() {
		res.Description = state.Description
	} else {
		res.Description = plan.Description
	}

	if plan.RbacConfig.IsUnknown() {
		res.RbacConfig = state.RbacConfig
	} else if plan.RbacConfig.IsNull() {
		res.RbacConfig = types.ObjectNull(RBACConfigMapAttrTypes)
	} else {
		var stateRbacConfig, planRbacConfig AkpArgoCDRBACConfig
		diags.Append(state.RbacConfig.As(context.Background(), &stateRbacConfig, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.RbacConfig.As(context.Background(), &planRbacConfig, basetypes.ObjectAsOptions{})...)
		resRbacConfig, d := MergeRbacConfig(&stateRbacConfig, &planRbacConfig)
		diags.Append(d...)
		res.RbacConfig, d = types.ObjectValueFrom(context.Background(), RBACConfigMapAttrTypes, resRbacConfig)
		diags.Append(d...)
	}

	if plan.Config.IsUnknown() {
		res.Config = plan.Config
	} else if plan.Config.IsNull() {
		res.Config = types.ObjectNull(configMapAttrTypes)
	} else {
		var stateConfig, planConfig AkpArgoCDConfig
		diags.Append(state.Config.As(context.Background(), &stateConfig, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.Config.As(context.Background(), &planConfig, basetypes.ObjectAsOptions{})...)
		resConfig, d := MergeConfig(&stateConfig, &planConfig)
		diags.Append(d...)
		res.Config, d = types.ObjectValueFrom(context.Background(), configMapAttrTypes, resConfig)
		diags.Append(d...)
	}

	if plan.Spec.IsUnknown() {
		res.Spec = plan.Spec
	} else if plan.Spec.IsNull() {
		res.Spec = types.ObjectNull(instanceSpecAttrTypes)
	} else {
		var stateSpec, planSpec AkpInstanceSpec
		diags.Append(state.Spec.As(context.Background(), &stateSpec, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.Spec.As(context.Background(), &planSpec, basetypes.ObjectAsOptions{})...)
		resSpec, d := MergeSpec(&stateSpec, &planSpec)
		diags.Append(d...)
		res.Spec, d = types.ObjectValueFrom(context.Background(), instanceSpecAttrTypes, resSpec)
		diags.Append(d...)
	}

	return res, diags
}

func (x *AkpInstance) UpdateFrom(p *argocdv1.Instance) diag.Diagnostics {
	diags := diag.Diagnostics{}
	d := diag.Diagnostics{}
	x.Id = types.StringValue(p.Id)
	x.Name = types.StringValue(p.Name)
	x.Version = types.StringValue(p.Version)
	x.Hostname = types.StringValue(p.Hostname)

	if p.Description == "" {
		x.Description = types.StringNull()
	} else {
		x.Description = types.StringValue(p.Description)
	}
	if p.RbacConfig == nil || p.RbacConfig.String() == "" {
		x.RbacConfig = types.ObjectNull(RBACConfigMapAttrTypes)
	} else {
		rbacConfig := &AkpArgoCDRBACConfig{}
		diags.Append(rbacConfig.UpdateObject(p.RbacConfig)...)
		x.RbacConfig, d = types.ObjectValueFrom(context.Background(), RBACConfigMapAttrTypes, rbacConfig)
		diags.Append(d...)
	}

	if p.Config == nil || p.Config.String() == "" {
		x.Config = types.ObjectNull(configMapAttrTypes)
	} else {
		config := &AkpArgoCDConfig{}
		diags.Append(config.UpdateObject(p.Config)...)
		x.Config, d = types.ObjectValueFrom(context.Background(), configMapAttrTypes, config)
		diags.Append(d...)
	}

	if p.Spec == nil || p.Spec.String() == "" {
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
		if target.RbacConfig != nil {
			diags.Append(rbacConfig.UpdateObject(target.RbacConfig)...)
		}
		targetRbacConfig := argocdv1.ArgoCDRBACConfigMap{}
		diags.Append(x.RbacConfig.As(context.Background(), &rbacConfig, basetypes.ObjectAsOptions{})...)
		diags.Append(rbacConfig.As(&targetRbacConfig)...)
		target.RbacConfig = &targetRbacConfig
	}

	if x.Config.IsNull() {
		target.Config = nil
	} else if !x.Config.IsUnknown() {
		config := AkpArgoCDConfig{}
		if target.Config != nil {
			diags.Append(config.UpdateObject(target.Config)...)
		}
		targetConfig := argocdv1.ArgoCDConfigMap{}
		diags.Append(x.Config.As(context.Background(), &config, basetypes.ObjectAsOptions{})...)
		diags.Append(config.As(&targetConfig)...)
		target.Config = &targetConfig
	}

	if x.Spec.IsNull() {
		target.Spec = nil
	} else if !x.Spec.IsUnknown() {
		spec := AkpInstanceSpec{}
		if target.Spec != nil {
			diags.Append(spec.UpdateObject(target.Spec)...)
		}
		targetSpec := argocdv1.InstanceSpec{}
		diags.Append(x.Spec.As(context.Background(), &spec, basetypes.ObjectAsOptions{})...)
		diags.Append(spec.As(&targetSpec)...)
		target.Spec = &targetSpec
	}
	return diags
}

package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpClusterCustomization struct {
	AutoUpgradeDisabled    types.Bool   `tfsdk:"auto_upgrade_disabled"`
	CustomRegistryArgoproj types.String `tfsdk:"custom_image_registry_argoproj"`
	CustomRegistryAkuity   types.String `tfsdk:"custom_image_registry_akuity"`
}

var (
	clusterCustomizationAttrTypes = map[string]attr.Type{
		"auto_upgrade_disabled":          types.BoolType,
		"custom_image_registry_argoproj": types.StringType,
		"custom_image_registry_akuity":   types.StringType,
	}
)

func MergeClusterCustomization(state *AkpClusterCustomization, plan *AkpClusterCustomization) (*AkpClusterCustomization, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpClusterCustomization{}

	if plan.AutoUpgradeDisabled.IsUnknown() {
		res.AutoUpgradeDisabled = state.AutoUpgradeDisabled
	} else if plan.AutoUpgradeDisabled.IsNull() {
		res.AutoUpgradeDisabled = types.BoolNull()
	} else {
		res.AutoUpgradeDisabled = plan.AutoUpgradeDisabled
	}

	if plan.CustomRegistryAkuity.IsUnknown() {
		res.CustomRegistryAkuity = state.CustomRegistryAkuity
	} else if plan.CustomRegistryAkuity.IsNull() {
		res.CustomRegistryAkuity = types.StringNull()
	} else {
		res.CustomRegistryAkuity = plan.CustomRegistryAkuity
	}

	if plan.CustomRegistryArgoproj.IsUnknown() {
		res.CustomRegistryArgoproj = state.CustomRegistryArgoproj
	} else if plan.CustomRegistryArgoproj.IsNull() {
		res.CustomRegistryArgoproj = types.StringNull()
	} else {
		res.CustomRegistryArgoproj = plan.CustomRegistryArgoproj
	}
	return res, diags
}

func (x *AkpClusterCustomization) UpdateObject(p *argocdv1.ClusterCustomization) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ClusterCustomization is <nil>")
		return diags
	}
	x.AutoUpgradeDisabled = types.BoolValue(p.GetAutoUpgradeDisabled())

	if p.CustomImageRegistryAkuity == "" {
		x.CustomRegistryAkuity = types.StringNull()
	} else {
		x.CustomRegistryAkuity = types.StringValue(p.CustomImageRegistryAkuity)
	}

	if p.CustomImageRegistryArgoproj == "" {
		x.CustomRegistryArgoproj = types.StringNull()
	} else {
		x.CustomRegistryArgoproj = types.StringValue(p.CustomImageRegistryArgoproj)
	}

	return diags
}

func (x *AkpClusterCustomization) As(target *argocdv1.ClusterCustomization) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.CustomImageRegistryAkuity = x.CustomRegistryAkuity.ValueString()
	target.CustomImageRegistryArgoproj = x.CustomRegistryArgoproj.ValueString()
	target.AutoUpgradeDisabled = x.AutoUpgradeDisabled.ValueBool()
	return diags
}

package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpClusterCustomization struct {
	CustomRegistryArgoproj types.String `tfsdk:"custom_image_registry_argoproj"`
	CustomRegistryAkuity   types.String `tfsdk:"custom_image_registry_akuity"`
	AutoUpgradeDisabled    types.Bool   `tfsdk:"auto_upgrade_disabled"`
}

var (
	clusterCustomizationAttrTypes = map[string]attr.Type{
		"custom_image_registry_argoproj": types.StringType,
		"custom_image_registry_akuity":   types.StringType,
		"auto_upgrade_disabled":          types.BoolType,
	}
)

func (x *AkpClusterCustomization) UpdateObject(p *argocdv1.ClusterCustomization) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ClusterCustomization is <nil>")
		return diags
	}
	x.CustomRegistryAkuity = types.StringValue(p.GetCustomImageRegistryAkuity())
	x.CustomRegistryArgoproj = types.StringValue(p.GetCustomImageRegistryArgoproj())
	x.AutoUpgradeDisabled = types.BoolValue(p.GetAutoUpgradeDisabled())
	return diags
}

func (x *AkpClusterCustomization) As(target *argocdv1.ClusterCustomization) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.CustomImageRegistryAkuity = x.CustomRegistryAkuity.ValueString()
	target.CustomImageRegistryArgoproj = x.CustomRegistryArgoproj.ValueString()
	target.AutoUpgradeDisabled = x.AutoUpgradeDisabled.ValueBool()
	return diags
}

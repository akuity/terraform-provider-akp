package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpClusterCustomization struct {
	AutoUpgradeDisabled    types.Bool   `tfsdk:"auto_upgrade_disabled"`
}

var (
	clusterCustomizationAttrTypes = map[string]attr.Type{
		"auto_upgrade_disabled":          types.BoolType,
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

	return res, diags
}

func (x *AkpClusterCustomization) UpdateObject(p *argocdv1.ClusterCustomization) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ClusterCustomization is <nil>")
		return diags
	}
	x.AutoUpgradeDisabled = types.BoolValue(p.GetAutoUpgradeDisabled())

	return diags
}

func (x *AkpClusterCustomization) As(target *argocdv1.ClusterCustomization) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.AutoUpgradeDisabled = x.AutoUpgradeDisabled.ValueBool()
	return diags
}

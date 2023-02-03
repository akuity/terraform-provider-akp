package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDKustomizeSettings struct {
	BuildOptions types.String `tfsdk:"build_options"`
	Enabled      types.Bool   `tfsdk:"enabled"`
}

var (
	kustomizeSettingsAttrTypes = map[string]attr.Type{
		"build_options": types.StringType,
		"enabled":       types.BoolType,
	}
)

func MergeKustomizeSettings(state *AkpArgoCDKustomizeSettings, plan *AkpArgoCDKustomizeSettings) (*AkpArgoCDKustomizeSettings, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpArgoCDKustomizeSettings{}

	if plan.BuildOptions.IsUnknown() {
		res.BuildOptions = state.BuildOptions
	} else if plan.BuildOptions.IsNull() {
		res.BuildOptions = types.StringNull()
	} else {
		res.BuildOptions = plan.BuildOptions
	}

	if plan.Enabled.IsUnknown() {
		res.Enabled = state.Enabled
	} else if plan.Enabled.IsNull() {
		res.Enabled = types.BoolNull()
	} else {
		res.Enabled = plan.Enabled
	}

	return res, diags
}

func (x *AkpArgoCDKustomizeSettings) UpdateObject(p *argocdv1.ArgoCDKustomizeSettings) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ArgoCDKustomizeSettings is <nil>")
		return diags
	}
	x.Enabled = types.BoolValue(p.GetEnabled())

	if p.BuildOptions == "" {
		x.BuildOptions = types.StringNull()
	} else {
		x.BuildOptions = types.StringValue(p.BuildOptions)
	}
	return diags
}

func (x *AkpArgoCDKustomizeSettings) As(target *argocdv1.ArgoCDKustomizeSettings) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Enabled = x.Enabled.ValueBool()
	target.BuildOptions = x.BuildOptions.ValueString()
	return diags
}

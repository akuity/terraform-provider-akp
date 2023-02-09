package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDKustomizeSettings struct {
	BuildOptions types.String `tfsdk:"build_options"`
}

var (
	KustomizeSettingsAttrTypes = map[string]attr.Type{
		"build_options": types.StringType,
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

	return res, diags
}

func (x *AkpArgoCDKustomizeSettings) UpdateObject(input *argocdv1.ArgoCDKustomizeSettings) diag.Diagnostics {
	diags := diag.Diagnostics{}
	var p *argocdv1.ArgoCDKustomizeSettings
	if input == nil {
		p = &argocdv1.ArgoCDKustomizeSettings{}
	} else {
		p = input
	}

	if p.BuildOptions == "" { // not computed
		x.BuildOptions = types.StringNull()
	} else {
		x.BuildOptions = types.StringValue(p.BuildOptions)
	}
	return diags
}

func (x *AkpArgoCDKustomizeSettings) As(target *argocdv1.ArgoCDKustomizeSettings) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.BuildOptions = x.BuildOptions.ValueString()
	return diags
}

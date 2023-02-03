package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDResourceSettings struct {
	CompareOptions types.String `tfsdk:"compare_options"`
	Exclusions     types.String `tfsdk:"exclusions"`
	Inclusions     types.String `tfsdk:"inclusions"`
}

var (
	resourceSettingsAttrTypes = map[string]attr.Type{
		"compare_options": types.StringType,
		"exclusions":      types.StringType,
		"inclusions":      types.StringType,
	}
)

func MergeResourceSettings(state *AkpArgoCDResourceSettings, plan *AkpArgoCDResourceSettings) (*AkpArgoCDResourceSettings, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpArgoCDResourceSettings{}

	if plan.CompareOptions.IsUnknown() {
		res.CompareOptions = state.CompareOptions
	} else if plan.CompareOptions.IsNull() {
		res.CompareOptions = types.StringNull()
	} else {
		res.CompareOptions = plan.CompareOptions
	}

	if plan.Exclusions.IsUnknown() {
		res.Exclusions = state.Exclusions
	} else if plan.Exclusions.IsNull() {
		res.Exclusions = types.StringNull()
	} else {
		res.Exclusions = plan.Exclusions
	}

	if plan.Inclusions.IsUnknown() {
		res.Inclusions = state.Inclusions
	} else if plan.Inclusions.IsNull() {
		res.Inclusions = types.StringNull()
	} else {
		res.Inclusions = plan.Inclusions
	}

	return res, diags
}

func (x *AkpArgoCDResourceSettings) UpdateObject(p *argocdv1.ArgoCDResourceSettings) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ArgoCDResourceSettings is <nil>")
		return diags
	}

	if p.CompareOptions == "" {
		x.CompareOptions = types.StringNull()
	} else {
		x.CompareOptions = types.StringValue(p.CompareOptions)
	}

	if p.Exclusions == "" {
		p.Exclusions = types.BoolType.String()
	} else {
		x.Exclusions = types.StringValue(p.Exclusions)
	}

	if p.Inclusions == "" {
		x.Inclusions = types.StringNull()
	} else {
		x.Inclusions = types.StringValue(p.Inclusions)
	}

	return diags
}

func (x *AkpArgoCDResourceSettings) As(target *argocdv1.ArgoCDResourceSettings) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Inclusions = x.Inclusions.ValueString()
	target.Exclusions = x.Exclusions.ValueString()
	target.CompareOptions = x.CompareOptions.ValueString()
	return diags
}

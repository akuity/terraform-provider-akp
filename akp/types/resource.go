package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDResourceSettings struct {
	Inclusions     types.String `tfsdk:"inclusions"`
	Exclusions     types.String `tfsdk:"exclusions"`
	CompareOptions types.String `tfsdk:"compare_options"`
}

var (
	resourceSettingsAttrTypes = map[string]attr.Type{
		"inclusions":      types.StringType,
		"exclusions":      types.StringType,
		"compare_options": types.StringType,
	}
)

func (x *AkpArgoCDResourceSettings) UpdateObject(p *argocdv1.ArgoCDResourceSettings) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ArgoCDResourceSettings is <nil>")
		return diags
	}
	x.Inclusions = types.StringValue(p.GetInclusions())
	x.Exclusions = types.StringValue(p.GetExclusions())
	x.CompareOptions = types.StringValue(p.GetCompareOptions())
	return diags
}

func (x *AkpArgoCDResourceSettings) As(target *argocdv1.ArgoCDResourceSettings) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Inclusions = x.Inclusions.ValueString()
	target.Exclusions = x.Exclusions.ValueString()
	target.CompareOptions = x.CompareOptions.ValueString()
	return diags
}

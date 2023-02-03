package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDStatusBadge struct {
	Enabled types.Bool   `tfsdk:"enabled"`
	Url     types.String `tfsdk:"url"`
}

var (
	statusBadgeAttrTypes = map[string]attr.Type{
		"enabled": types.BoolType,
		"url":     types.StringType,
	}
)

func MergeStatusBadge(state *AkpArgoCDStatusBadge, plan *AkpArgoCDStatusBadge) (*AkpArgoCDStatusBadge, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpArgoCDStatusBadge{}

	if plan.Enabled.IsUnknown() {
		res.Enabled = state.Enabled
	} else if plan.Enabled.IsNull() {
		res.Enabled = types.BoolNull()
	} else {
		res.Enabled = plan.Enabled
	}

	if plan.Url.IsUnknown() {
		res.Url = state.Url
	} else if plan.Url.IsNull() {
		res.Url = types.StringNull()
	} else {
		res.Url = plan.Url
	}

	return res, diags
}

func (x *AkpArgoCDStatusBadge) UpdateObject(p *argocdv1.ArgoCDStatusBadgeConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ArgoCDStatusBadgeConfig is <nil>")
		return diags
	}
	x.Enabled = types.BoolValue(p.GetEnabled())

	if p.Url == "" {
		x.Url = types.StringNull()
	} else {
		x.Url = types.StringValue(p.Url)
	}
	return diags
}

func (x *AkpArgoCDStatusBadge) As(target *argocdv1.ArgoCDStatusBadgeConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Enabled = x.Enabled.ValueBool()
	target.Url = x.Url.ValueString()
	return diags
}

package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDBanner struct {
	Message   types.String `tfsdk:"message"`
	Permanent types.Bool   `tfsdk:"permanent"`
	Url       types.String `tfsdk:"url"`
}

var (
	bannerAttrTypes = map[string]attr.Type{
		"message":   types.StringType,
		"permanent": types.BoolType,
		"url":       types.StringType,
	}
)

func MergeBanner(state *AkpArgoCDBanner, plan *AkpArgoCDBanner) (*AkpArgoCDBanner, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpArgoCDBanner{}

	if plan.Message.IsUnknown() {
		res.Message = state.Message
	} else if plan.Message.IsNull() {
		res.Message = types.StringNull()
	} else {
		res.Message = plan.Message
	}

	if plan.Permanent.IsUnknown() {
		res.Permanent = state.Permanent
	} else if plan.Permanent.IsNull() {
		res.Permanent = types.BoolNull()
	} else {
		res.Permanent = plan.Permanent
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

func (x *AkpArgoCDBanner) UpdateObject(p *argocdv1.ArgoCDBannerConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ArgoCDBannerConfig is <nil>")
		return diags
	}
	if p.Message == "" {
		x.Message = types.StringNull()
	} else {
		x.Message = types.StringValue(p.Message)
	}

	x.Permanent = types.BoolValue(p.GetPermanent())

	if p.Url == "" {
		x.Url = types.StringNull()
	} else {
		x.Url = types.StringValue(p.Url)
	}

	return diags
}

func (x *AkpArgoCDBanner) As(target *argocdv1.ArgoCDBannerConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Message = x.Message.ValueString()
	target.Url = x.Url.ValueString()
	target.Permanent = x.Permanent.ValueBool()
	return diags
}

package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDBanner struct {
	Message   types.String `tfsdk:"message"`
	Url       types.String `tfsdk:"url"`
	Permanent types.Bool   `tfsdk:"permanent"`
}

var (
	bannerAttrTypes = map[string]attr.Type{
		"message":   types.StringType,
		"url":       types.StringType,
		"permanent": types.BoolType,
	}
)

func (x *AkpArgoCDBanner) UpdateObject(p *argocdv1.ArgoCDBannerConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	x.Message = types.StringValue(p.GetMessage())
	x.Url = types.StringValue(p.GetUrl())
	x.Permanent = types.BoolValue(p.GetPermanent())
	return diags
}

func (x *AkpArgoCDBanner) As(target *argocdv1.ArgoCDBannerConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Message = x.Message.ValueString()
	target.Url = x.Url.ValueString()
	target.Permanent = x.Permanent.ValueBool()
	return diags
}

package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDChat struct {
	Message types.String `tfsdk:"message"`
	Url     types.String `tfsdk:"url"`
}

var (
	chatAttrTypes = map[string]attr.Type{
		"message": types.StringType,
		"url":     types.StringType,
	}
)

func MergeChat(state *AkpArgoCDChat, plan *AkpArgoCDChat) (*AkpArgoCDChat, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpArgoCDChat{}

	if plan.Message.IsUnknown() {
		res.Message = state.Message
	} else if plan.Message.IsNull() {
		res.Message = types.StringNull()
	} else {
		res.Message = plan.Message
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

func (x *AkpArgoCDChat) UpdateObject(input *argocdv1.ArgoCDAlertConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	var p *argocdv1.ArgoCDAlertConfig
	if input == nil {
		p = &argocdv1.ArgoCDAlertConfig{}
	} else {
		p = input
	}
	if p.Message == "" {
		x.Message = types.StringNull()
	} else {
		x.Message = types.StringValue(p.Message)
	}

	if p.Url == "" {
		x.Url = types.StringNull()
	} else {
		x.Url = types.StringValue(p.Url)
	}

	return diags
}

func (x *AkpArgoCDChat) As(target *argocdv1.ArgoCDAlertConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Message = x.Message.ValueString()
	target.Url = x.Url.ValueString()
	return diags
}

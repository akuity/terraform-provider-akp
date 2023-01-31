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

func (x *AkpArgoCDChat) UpdateObject(p *argocdv1.ArgoCDAlertConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	x.Message = types.StringValue(p.GetMessage())
	x.Url = types.StringValue(p.GetUrl())
	return diags
}

func (x *AkpArgoCDChat) As(target *argocdv1.ArgoCDAlertConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Message = x.Message.ValueString()
	target.Url = x.Url.ValueString()
	return diags
}

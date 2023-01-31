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

func (x *AkpArgoCDStatusBadge) UpdateObject(p *argocdv1.ArgoCDStatusBadgeConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	x.Enabled = types.BoolValue(p.GetEnabled())
	x.Url = types.StringValue(p.GetUrl())
	return diags
}

func (x *AkpArgoCDStatusBadge) As(target *argocdv1.ArgoCDStatusBadgeConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Enabled = x.Enabled.ValueBool()
	target.Url = x.Url.ValueString()
	return diags
}

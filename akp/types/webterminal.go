package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDWebTerminal struct {
	Enabled types.Bool   `tfsdk:"enabled"`
	Shells  types.String `tfsdk:"shells"`
}

var (
	webTerminalAttrTypes = map[string]attr.Type{
		"enabled": types.BoolType,
		"shells": types.StringType,
	}
)

func (x *AkpArgoCDWebTerminal) UpdateObject(p *argocdv1.ArgoCDWebTerminalConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	x.Enabled = types.BoolValue(p.GetEnabled())
	x.Shells = types.StringValue(p.GetShells())
	return diags
}

func (x *AkpArgoCDWebTerminal) As(target *argocdv1.ArgoCDWebTerminalConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Enabled = x.Enabled.ValueBool()
	target.Shells = x.Shells.ValueString()
	return diags
}

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
		"shells":  types.StringType,
	}
)

func MergeWebTerminal(state *AkpArgoCDWebTerminal, plan *AkpArgoCDWebTerminal) (*AkpArgoCDWebTerminal, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpArgoCDWebTerminal{}

	if plan.Enabled.IsUnknown() {
		res.Enabled = state.Enabled
	} else if plan.Enabled.IsNull() {
		res.Enabled = types.BoolNull()
	} else {
		res.Enabled = plan.Enabled
	}

	if plan.Shells.IsUnknown() {
		res.Shells = state.Shells
	} else if plan.Shells.IsNull() {
		res.Shells = types.StringNull()
	} else {
		res.Shells = plan.Shells
	}

	return res, diags
}

func (x *AkpArgoCDWebTerminal) UpdateObject(p *argocdv1.ArgoCDWebTerminalConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ArgoCDWebTerminalConfig is <nil>")
		return diags
	}
	x.Enabled = types.BoolValue(p.GetEnabled())
	if p.Shells == "" {
		x.Shells = types.StringNull()
	} else {
		x.Shells = types.StringValue(p.Shells)
	}
	return diags
}

func (x *AkpArgoCDWebTerminal) As(target *argocdv1.ArgoCDWebTerminalConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Enabled = x.Enabled.ValueBool()
	target.Shells = x.Shells.ValueString()
	return diags
}

package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDKustomizeSettings struct {
	Enabled      types.Bool   `tfsdk:"enabled"`
	BuildOptions types.String `tfsdk:"build_options"`
}

var (
	kustomizeSettingsAttrTypes = map[string]attr.Type{
		"enabled":       types.BoolType,
		"build_options": types.StringType,
	}
)

func (x *AkpArgoCDKustomizeSettings) UpdateObject(p *argocdv1.ArgoCDKustomizeSettings) diag.Diagnostics {
	diags := diag.Diagnostics{}
	x.Enabled = types.BoolValue(p.GetEnabled())
	x.BuildOptions = types.StringValue(p.GetBuildOptions())
	return diags
}

func (x *AkpArgoCDKustomizeSettings) As(target *argocdv1.ArgoCDKustomizeSettings) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Enabled = x.Enabled.ValueBool()
	target.BuildOptions = x.BuildOptions.ValueString()
	return diags
}

package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDHelmSettings struct {
	Enabled          types.Bool   `tfsdk:"enabled"`
	ValueFileSchemas types.String `tfsdk:"value_file_schemas"`
}

var (
	helmSettingsAttrTypes = map[string]attr.Type{
		"enabled":            types.BoolType,
		"value_file_schemas": types.StringType,
	}
)

func MergeHelmSettings(state *AkpArgoCDHelmSettings, plan *AkpArgoCDHelmSettings) (*AkpArgoCDHelmSettings, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpArgoCDHelmSettings{}

	if plan.Enabled.IsUnknown() {
		res.Enabled = state.Enabled
	} else if plan.Enabled.IsNull() {
		res.Enabled = types.BoolNull()
	} else {
		res.Enabled = plan.Enabled
	}

	if plan.ValueFileSchemas.IsUnknown() {
		res.ValueFileSchemas = state.ValueFileSchemas
	} else if plan.ValueFileSchemas.IsNull() {
		res.ValueFileSchemas = types.StringNull()
	} else {
		res.ValueFileSchemas = plan.ValueFileSchemas
	}

	return res, diags
}

func (x *AkpArgoCDHelmSettings) UpdateObject(p *argocdv1.ArgoCDHelmSettings) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ArgoCDHelmSettings is <nil>")
		return diags
	}
	x.Enabled = types.BoolValue(p.GetEnabled())

	if p.ValueFileSchemas == "" {
		x.ValueFileSchemas = types.StringNull()
	} else {
		x.ValueFileSchemas = types.StringValue(p.ValueFileSchemas)
	}
	return diags
}

func (x *AkpArgoCDHelmSettings) As(target *argocdv1.ArgoCDHelmSettings) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Enabled = x.Enabled.ValueBool()
	target.ValueFileSchemas = x.ValueFileSchemas.ValueString()
	return diags
}

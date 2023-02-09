package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDHelmSettings struct {
	ValueFileSchemas types.String `tfsdk:"value_file_schemas"`
}

var (
	HelmSettingsAttrTypes = map[string]attr.Type{
		"value_file_schemas": types.StringType,
	}
)

func MergeHelmSettings(state *AkpArgoCDHelmSettings, plan *AkpArgoCDHelmSettings) (*AkpArgoCDHelmSettings, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpArgoCDHelmSettings{}
	if plan.ValueFileSchemas.IsUnknown() {
		res.ValueFileSchemas = state.ValueFileSchemas
	} else if plan.ValueFileSchemas.IsNull() {
		res.ValueFileSchemas = types.StringNull()
	} else {
		res.ValueFileSchemas = plan.ValueFileSchemas
	}

	return res, diags
}

func (x *AkpArgoCDHelmSettings) UpdateObject(input *argocdv1.ArgoCDHelmSettings) diag.Diagnostics {
	diags := diag.Diagnostics{}
	var p *argocdv1.ArgoCDHelmSettings
	if input == nil {
		p = &argocdv1.ArgoCDHelmSettings{}
	} else {
		p = input
	}

	if p.ValueFileSchemas == "" { // not computed
		x.ValueFileSchemas = types.StringNull()
	} else {
		x.ValueFileSchemas = types.StringValue(p.ValueFileSchemas)
	}
	return diags
}

func (x *AkpArgoCDHelmSettings) As(target *argocdv1.ArgoCDHelmSettings) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.ValueFileSchemas = x.ValueFileSchemas.ValueString()
	return diags
}

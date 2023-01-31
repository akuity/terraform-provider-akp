package types

import (
	"context"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDRBACConfig struct {
	DefaultPolicy types.String `tfsdk:"default_policy"`
	PolicyCsv     types.String `tfsdk:"policy_csv"`
	Scopes        types.List   `tfsdk:"scopes"`
}

var (
	RBACConfigMapAttrTypes = map[string]attr.Type{
		"default_policy": types.StringType,
		"policy_csv":     types.StringType,
		"scopes": types.ListType{
			ElemType: types.StringType,
		},
	}
)

func (x *AkpArgoCDRBACConfig) UpdateObject(p *argocdv1.ArgoCDRBACConfigMap) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ArgoCDRBACConfigMap is <nil>")
		return diags
	}
	x.DefaultPolicy = types.StringValue(p.GetDefaultPolicy())
	x.PolicyCsv = types.StringValue(p.GetPolicyCsv())
	x.Scopes, diags = types.ListValueFrom(context.Background(), types.StringType, p.GetScopes())
	return diags
}

func (x *AkpArgoCDRBACConfig) As(target *argocdv1.ArgoCDRBACConfigMap) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.DefaultPolicy = x.DefaultPolicy.ValueString()
	target.PolicyCsv = x.PolicyCsv.ValueString()
	if x.Scopes.IsNull() {
		target.Scopes = nil
	} else if !x.Scopes.IsUnknown() {
		var scopes []string
		diags.Append(x.Scopes.ElementsAs(context.Background(), &scopes, true)...)
		target.Scopes = scopes
	}
	return diags
}

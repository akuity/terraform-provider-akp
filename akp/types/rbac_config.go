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

func MergeRbacConfig(state *AkpArgoCDRBACConfig, plan *AkpArgoCDRBACConfig) (*AkpArgoCDRBACConfig, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpArgoCDRBACConfig{}

	if plan.DefaultPolicy.IsUnknown() {
		res.DefaultPolicy = state.DefaultPolicy
	} else if plan.DefaultPolicy.IsNull() {
		res.DefaultPolicy = types.StringNull()
	} else {
		res.DefaultPolicy = plan.DefaultPolicy
	}

	if plan.PolicyCsv.IsUnknown() {
		res.PolicyCsv = state.PolicyCsv
	} else if plan.PolicyCsv.IsNull() {
		res.PolicyCsv = types.StringNull()
	} else {
		res.PolicyCsv = plan.PolicyCsv
	}

	if plan.Scopes.IsUnknown() {
		res.Scopes = state.Scopes
	} else if plan.Scopes.IsNull() {
		res.Scopes = types.ListNull(types.StringType)
	} else {
		res.Scopes = plan.Scopes
	}

	return res, diags
}

func (x *AkpArgoCDRBACConfig) UpdateObject(p *argocdv1.ArgoCDRBACConfigMap) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ArgoCDRBACConfigMap is <nil>")
		return diags
	}
	if p.DefaultPolicy == "" {
		x.DefaultPolicy = types.StringNull()
	} else {
		x.DefaultPolicy = types.StringValue(p.DefaultPolicy)
	}

	if p.PolicyCsv == "" {
		x.PolicyCsv = types.StringNull()
	} else {
		x.PolicyCsv = types.StringValue(p.PolicyCsv)
	}
	if len(p.Scopes) == 0 {
		x.Scopes = types.ListNull(types.StringType)
	} else {
		var scopes []types.String
		for _, entry := range p.Scopes {
			scope := types.StringValue(entry)
			scopes = append(scopes, scope)
		}
		x.Scopes, diags = types.ListValueFrom(context.Background(), types.StringType, scopes)
	}
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

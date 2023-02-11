package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Secret struct {
	Value types.String `tfsdk:"value"`
}

var (
	secretAttrTypes = map[string]attr.Type{
		"value": types.StringType,
	}
)

func MapValueFromMap(s map[string]string) (types.Map, diag.Diagnostics) {
	var res types.Map
	var diags diag.Diagnostics
	secrets := make(map[string]Secret)
	for name, value := range s {
		secrets[name] = Secret{
			Value: types.StringValue(value),
		}
	}
	if len(secrets) == 0 {
		res = types.MapNull(types.ObjectType{AttrTypes: secretAttrTypes})
	} else {
		res, diags = types.MapValueFrom(context.Background(), types.ObjectType{AttrTypes: secretAttrTypes}, secrets)
	}
	return res, diags
}

func MapFromMapValue(s types.Map) (map[string]string, diag.Diagnostics) {
	var secrets map[string]Secret
	d := s.ElementsAs(context.Background(), &secrets, true)
	res := make(map[string]string)
	for name, elem := range secrets {
		res[name] = elem.Value.ValueString()
	}
	return res, d
}

func MergeSecrets(state *types.Map, plan *types.Map) (*types.Map, diag.Diagnostics) {
	var stateSecrets, planSecrets map[string]Secret
	diags := diag.Diagnostics{}
	if !state.IsNull() {
		diags.Append(state.ElementsAs(context.Background(), &stateSecrets, true)...)
	} else {
		stateSecrets = make(map[string]Secret)
	}
	if !plan.IsNull() && !plan.IsUnknown() {
		diags.Append(plan.ElementsAs(context.Background(), &planSecrets, true)...)
	} else {
		planSecrets = make(map[string]Secret)
	}
	res := make(map[string]Secret)
	for name := range stateSecrets {
		if val, ok := planSecrets[name]; ok {
			res[name] = val // update secret from plan
		} else {
			res[name] = Secret{
				Value: types.StringNull(), // remove secret
			}
		}
	}
	for name := range planSecrets {
		if _, ok := stateSecrets[name]; !ok {
			res[name] = planSecrets[name]
		}
	}
	resMap, d := types.MapValueFrom(context.Background(), types.ObjectType{AttrTypes: secretAttrTypes}, res)
	return &resMap, d
}

package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Secret struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

var (
	secretAttrTypes = map[string]attr.Type{
		"name":  types.StringType,
		"value": types.StringType,
	}
)

func ListValueFromMap(s map[string]string) (types.List, diag.Diagnostics) {
	var secrets []Secret
	for name, value := range s {
		secret := Secret{
			Name:  types.StringValue(name),
			Value: types.StringValue(value),
		}
		secrets = append(secrets, secret)
	}
	res, d := types.ListValueFrom(context.Background(), types.ObjectType{AttrTypes: secretAttrTypes}, secrets)
	return res, d
}

func MapFromListValue(s types.List) (map[string]string, diag.Diagnostics) {
	var secrets []Secret
	d := s.ElementsAs(context.Background(),&secrets, true)
	res := make(map[string]string)
	for _, elem := range secrets {
		res[elem.Name.ValueString()] = elem.Value.ValueString()
	}
	return res, d
}

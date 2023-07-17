package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Secret struct {
	SecretObjectMeta `json:"metadata" tfsdk:"metadata"`
	Data             types.Map    `json:"data,omitempty" tfsdk:"data"`
	StringData       types.Map    `json:"stringData,omitempty" tfsdk:"string_data"`
	Type             types.String `json:"type,omitempty" tfsdk:"type"`
}

func (s *Secret) GetSensitiveStrings() []string {
	var res []string
	secrets, _ := MapFromMapValue(s.Data)
	for _, value := range secrets {
		res = append(res, value)
	}
	secrets, _ = MapFromMapValue(s.StringData)
	for _, value := range secrets {
		res = append(res, value)
	}
	return res
}

func MapFromMapValue(s types.Map) (map[string]string, diag.Diagnostics) {
	var data map[string]string
	var d diag.Diagnostics
	if !s.IsNull() {
		d = s.ElementsAs(context.Background(), &data, true)
	}
	return data, d
}

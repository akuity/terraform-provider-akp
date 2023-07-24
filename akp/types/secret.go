package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Secret struct {
	Name       types.String `tfsdk:"name"`
	Labels     types.Map    `tfsdk:"labels"`
	Data       types.Map    `tfsdk:"data"`
	StringData types.Map    `tfsdk:"string_data"`
	Type       types.String `tfsdk:"type"`
}

func (s *Secret) GetSensitiveStrings() []string {
	var res []string
	if s == nil {
		return res
	}
	secrets, _ := mapFromMapValue(s.Data)
	for _, value := range secrets {
		res = append(res, value)
	}
	secrets, _ = mapFromMapValue(s.StringData)
	for _, value := range secrets {
		res = append(res, value)
	}
	return res
}

func (s *Secret) ToSecretAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, name string) *v1.Secret {
	var labels map[string]string
	var data map[string][]byte
	var stringData map[string]string
	diagnostics.Append(s.Labels.ElementsAs(ctx, &labels, true)...)
	diagnostics.Append(s.Data.ElementsAs(ctx, &data, true)...)
	diagnostics.Append(s.StringData.ElementsAs(ctx, &stringData, true)...)
	n := name
	if !s.Name.IsNull() && !s.Name.IsUnknown() {
		n = s.Name.ValueString()
	}
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   n,
			Labels: labels,
		},
		Data:       data,
		StringData: stringData,
	}
}

func mapFromMapValue(s types.Map) (map[string]string, diag.Diagnostics) {
	var data map[string]string
	var d diag.Diagnostics
	if !s.IsNull() {
		d = s.ElementsAs(context.Background(), &data, true)
	}
	return data, d
}

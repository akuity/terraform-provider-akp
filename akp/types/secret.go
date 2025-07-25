package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetSensitiveStrings(data tftypes.Map) []string {
	var res []string
	if data.IsNull() || data.IsUnknown() {
		return res
	}
	secrets, _ := mapFromMapValue(data)
	for _, value := range secrets {
		res = append(res, value)
	}
	return res
}

func ToSecretAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, name string, labels map[string]string, m tftypes.Map) *v1.Secret {
	var data map[string]string
	diagnostics.Append(m.ElementsAs(ctx, &data, true)...)
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		StringData: data,
	}
}

func mapFromMapValue(s tftypes.Map) (map[string]string, diag.Diagnostics) {
	var data map[string]string
	var d diag.Diagnostics
	if !s.IsNull() {
		d = s.ElementsAs(context.Background(), &data, true)
	}
	return data, d
}

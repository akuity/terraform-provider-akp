package types

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/akuity/terraform-provider-akp/akp/marshal"
)

type ConfigMap struct {
	Data tftypes.Map `json:"data,omitempty" tfsdk:"data"`
}

func (c *ConfigMap) Update(ctx context.Context, diagnostics *diag.Diagnostics, data *structpb.Struct) {
	m := map[string]string{}
	err := marshal.RemarshalTo(data, &m)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Argo CD instance. %s", err))
	}
	newData, diag := tftypes.MapValueFrom(ctx, tftypes.StringType, &m)
	diagnostics.Append(diag...)
	c.Data = newData
}

func (c *ConfigMap) ToConfigMapAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, name string) *v1.ConfigMap {
	var data map[string]string
	diagnostics.Append(c.Data.ElementsAs(ctx, &data, true)...)
	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: data,
	}
}

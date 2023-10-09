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

func ToConfigMapTFModel(ctx context.Context, diagnostics *diag.Diagnostics, data *structpb.Struct, oldCM tftypes.Map) tftypes.Map {
	if data == nil || len(data.AsMap()) == 0 {
		if !oldCM.IsUnknown() && (oldCM.IsNull() || len(oldCM.Elements()) == 0) {
			return oldCM
		}
	}
	m := map[string]string{}
	err := marshal.RemarshalTo(data, &m)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get ConfigMap data. %s", err))
		return tftypes.MapNull(tftypes.StringType)
	}
	newData, diag := tftypes.MapValueFrom(ctx, tftypes.StringType, &m)
	diagnostics.Append(diag...)
	return newData
}

func ToConfigMapAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, name string, m tftypes.Map) *v1.ConfigMap {
	var data map[string]string
	diagnostics.Append(m.ElementsAs(ctx, &data, true)...)
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

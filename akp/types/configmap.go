package types

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/structpb"
	v1 "k8s.io/api/core/v1"

	"github.com/akuity/terraform-provider-akp/akp/marshal"
)

type ConfigMap struct {
	Data tftypes.Map `json:"data,omitempty" tfsdk:"data"`
}

func (cm *ConfigMap) Update(ctx context.Context, diagnostics *diag.Diagnostics, data *structpb.Struct) {
	m := map[string]string{}
	err := marshal.RemarshalTo(data, &m)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Argo CD instance. %s", err))
	}
	configMap := &v1.ConfigMap{
		Data: m,
	}
	newCM := ToConfigMapTFModel(ctx, diagnostics, configMap)
	cm.Data = mergeStringMaps(ctx, diagnostics, cm.Data, newCM.Data)
}

func mergeStringMaps(ctx context.Context, diagnostics *diag.Diagnostics, old, new tftypes.Map) tftypes.Map {
	var oldData, newData map[string]string
	if !new.IsNull() {
		diagnostics.Append(new.ElementsAs(ctx, &newData, true)...)
	} else {
		newData = make(map[string]string)
	}
	if !old.IsNull() {
		diagnostics.Append(old.ElementsAs(ctx, &oldData, true)...)
	} else {
		oldData = make(map[string]string)
	}
	res := make(map[string]string)
	for name := range oldData {
		if val, ok := newData[name]; ok {
			res[name] = val
		} else {
			delete(res, name)
		}
	}
	resMap, d := tftypes.MapValueFrom(ctx, tftypes.StringType, res)
	diagnostics.Append(d...)
	return resMap
}

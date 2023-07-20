package types

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/akuity/terraform-provider-akp/akp/marshal"
)

type ConfigMap struct {
	Data tftypes.Map `json:"data,omitempty" tfsdk:"data"`
}

func Update(ctx context.Context, diagnostics *diag.Diagnostics, cm *ConfigMap, data *structpb.Struct) *ConfigMap {
	if cm == nil {
		cm = &ConfigMap{}
	}
	m := map[string]string{}
	err := marshal.RemarshalTo(data, &m)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Argo CD instance. %s", err))
	}
	newData, diag := tftypes.MapValueFrom(ctx, tftypes.StringType, &m)
	diagnostics.Append(diag...)
	cm.Data = mergeStringMaps(ctx, diagnostics, cm.Data, newData)
	return cm
}

func mergeStringMaps(ctx context.Context, diagnostics *diag.Diagnostics, old, new tftypes.Map) tftypes.Map {
	var oldData, newData map[string]string
	if !new.IsNull() {
		diagnostics.Append(new.ElementsAs(ctx, &newData, true)...)
	} else {
		newData = make(map[string]string)
	}
	if old.IsNull() {
		return new
	}

	diagnostics.Append(old.ElementsAs(ctx, &oldData, true)...)
	tflog.Info(ctx, fmt.Sprintf("-----------new data:%+v old data: %+v", newData, oldData))
	res := make(map[string]string)
	for name := range oldData {
		if val, ok := newData[name]; ok {
			res[name] = val
		}
	}
	resMap, d := tftypes.MapValueFrom(ctx, tftypes.StringType, res)
	tflog.Info(ctx, fmt.Sprintf("-----------res map: %+v", resMap))
	diagnostics.Append(d...)
	return resMap
}

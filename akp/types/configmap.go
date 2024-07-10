package types

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ToConfigMapTFModel(ctx context.Context, diagnostics *diag.Diagnostics, data *structpb.Struct, oldCM tftypes.Map) tftypes.Map {
	if data == nil || len(data.AsMap()) == 0 {
		if !oldCM.IsUnknown() && (oldCM.IsNull() || len(oldCM.Elements()) == 0) {
			return oldCM
		}
	}
	m := data.AsMap()
	for k, v := range m {
		switch t := v.(type) {
		case string:
			sortedValue, err := sortJSONString(t)
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to sort JSON keys for key %s. %s", k, err))
				return tftypes.MapNull(tftypes.StringType)
			}
			m[k] = sortedValue
		default:
			m[k] = v
		}
	}

	newData, diag := tftypes.MapValueFrom(ctx, tftypes.StringType, &m)
	diagnostics.Append(diag...)
	return newData
}

func ToConfigMapAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, name string, m tftypes.Map) *v1.ConfigMap {
	var data map[string]string
	diagnostics.Append(m.ElementsAs(ctx, &data, true)...)
	for k, v := range data {
		if json.Valid([]byte(v)) {
			sortedValue, err := sortJSONString(v)
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to sort JSON keys for key %s. %s", k, err))
				return nil
			}
			data[k] = sortedValue
		}
	}
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

func sortJSONKeys(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case map[string]interface{}:
		sortedMap := make(map[string]interface{})
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sortedValue, err := sortJSONKeys(v[k])
			if err != nil {
				return nil, err
			}
			sortedMap[k] = sortedValue
		}
		return sortedMap, nil
	case []interface{}:
		sortedArray := make([]interface{}, len(v))
		for i, item := range v {
			sortedValue, err := sortJSONKeys(item)
			if err != nil {
				return nil, err
			}
			sortedArray[i] = sortedValue
		}
		return sortedArray, nil
	default:
		return v, nil
	}
}

func sortJSONString(jsonStr string) (string, error) {
	if !json.Valid([]byte(jsonStr)) {
		return jsonStr, nil
	}

	var data interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return "", err
	}

	sortedData, err := sortJSONKeys(data)
	if err != nil {
		return "", err
	}

	sortedJSON, err := json.Marshal(sortedData)
	if err != nil {
		return "", err
	}

	return string(sortedJSON), nil
}

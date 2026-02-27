package types

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func ToFilteredConfigMapTFModel(ctx context.Context, diagnostics *diag.Diagnostics, data *structpb.Struct, oldCM tftypes.Map) tftypes.Map {
	if data == nil || len(data.AsMap()) == 0 {
		if !oldCM.IsUnknown() && (oldCM.IsNull() || len(oldCM.Elements()) == 0) {
			return oldCM
		}
	}

	oldMap := make(map[string]interface{}, len(oldCM.Elements()))
	for k, v := range oldCM.Elements() {
		oldMap[k] = v
	}

	m := data.AsMap()

	mergedCustomizations := parseMergedResourceCustomizations(m)

	// Only include values which are a part of the original resource map. The reason for doing so is that the API returns
	// a lot of fields which can cause TF to have an inconsistent state. We rely on the backend being able to do the right
	// thing in regard to PATCH requests; we don't actually need to have all the fields which the API returns in the state.
	for k := range oldMap {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				if k == "resource.customizations" {
					if yamlSemanticEqual(oldMap[k], t) {
						continue
					}
				}
				sortedValue, err := sortJSONString(t)
				if err != nil {
					diagnostics.AddError("Client Error", fmt.Sprintf("Unable to sort JSON keys for key %s. %s", k, err))
					return tftypes.MapNull(tftypes.StringType)
				}
				oldMap[k] = sortedValue
			default:
				oldMap[k] = v
			}
		} else if v, ok := resolveResourceCustomizationKey(k, mergedCustomizations); ok {
			oldMap[k] = v
		}
	}

	newData, diag := tftypes.MapValueFrom(ctx, tftypes.StringType, &oldMap)
	diagnostics.Append(diag...)
	return newData
}

func parseMergedResourceCustomizations(apiMap map[string]any) map[string]string {
	result := make(map[string]string)

	raw, ok := apiMap["resource.customizations"]
	if !ok {
		return result
	}
	yamlStr, ok := raw.(string)
	if !ok {
		return result
	}

	var customizations map[string]map[string]any
	if err := yaml.Unmarshal([]byte(yamlStr), &customizations); err != nil {
		return result
	}

	for groupKind, fields := range customizations {
		flatGroupKind := strings.ReplaceAll(groupKind, "/", "_")
		for fieldName, fieldValue := range fields {
			var valueStr string
			switch v := fieldValue.(type) {
			case string:
				valueStr = strings.TrimSpace(v)
			default:
				data, err := yaml.Marshal(v)
				if err != nil {
					continue
				}
				valueStr = strings.TrimSpace(string(data))
			}
			key := fmt.Sprintf("resource.customizations.%s.%s", fieldName, flatGroupKind)
			result[key] = valueStr
		}
	}

	return result
}

func yamlSemanticEqual(oldVal interface{}, newYAML string) bool {
	var oldStr string
	switch v := oldVal.(type) {
	case string:
		oldStr = v
	case tftypes.String:
		if v.IsNull() || v.IsUnknown() {
			return false
		}
		oldStr = v.ValueString()
	default:
		return false
	}

	var oldMap, newMap interface{}
	if err := yaml.Unmarshal([]byte(oldStr), &oldMap); err != nil {
		return false
	}
	if err := yaml.Unmarshal([]byte(newYAML), &newMap); err != nil {
		return false
	}
	return reflect.DeepEqual(oldMap, newMap)
}

func resolveResourceCustomizationKey(key string, mergedCustomizations map[string]string) (string, bool) {
	if !strings.HasPrefix(key, "resource.customizations.") {
		return "", false
	}
	v, ok := mergedCustomizations[key]
	return v, ok
}

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

func sortJSONKeys(value any) (any, error) {
	switch v := value.(type) {
	case map[string]any:
		sortedMap := make(map[string]any)
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
	case []any:
		sortedArray := make([]any, len(v))
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

	var data any
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return "", err
	}

	switch data.(type) {
	case map[string]any, []any:
	default:
		return jsonStr, nil
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

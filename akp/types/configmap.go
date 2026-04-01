package types

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func FilterMapToPlannedKeys(_ context.Context, diagnostics *diag.Diagnostics, current, planned tftypes.Map) tftypes.Map {
	if planned.IsNull() || planned.IsUnknown() || current.IsNull() || current.IsUnknown() {
		return current
	}

	plannedElems := planned.Elements()
	currentElems := current.Elements()

	filtered := make(map[string]attr.Value, len(plannedElems))
	for k := range plannedElems {
		if v, ok := currentElems[k]; ok {
			filtered[k] = v
		}
	}

	result, d := tftypes.MapValue(tftypes.StringType, filtered)
	diagnostics.Append(d...)
	return result
}

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
		oldValue := oldMap[k]
		oldString, hasOldString := configMapStringValue(oldValue)
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				if k == "resource.customizations" {
					if yamlIsSubset(t, oldMap[k]) {
						continue
					}
				}
				if hasOldString && equivalentConfigMapString(k, oldString, t) {
					oldMap[k] = oldValue
					continue
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
			if hasOldString && equivalentConfigMapString(k, oldString, v) {
				oldMap[k] = oldValue
				continue
			}
			oldMap[k] = v
		}
	}

	newData, diag := tftypes.MapValueFrom(ctx, tftypes.StringType, &oldMap)
	diagnostics.Append(diag...)
	return newData
}

func configMapStringValue(val any) (string, bool) {
	switch v := val.(type) {
	case string:
		return v, true
	case tftypes.String:
		if v.IsNull() || v.IsUnknown() {
			return "", false
		}
		return v.ValueString(), true
	default:
		return "", false
	}
}

func equivalentConfigMapString(key, oldValue, newValue string) bool {
	if isAccountCapabilitiesKey(key) {
		return normalizeAccountCapabilities(oldValue) == normalizeAccountCapabilities(newValue)
	}
	if json.Valid([]byte(oldValue)) && json.Valid([]byte(newValue)) {
		sortedOld, err := sortJSONString(oldValue)
		if err != nil {
			return false
		}
		sortedNew, err := sortJSONString(newValue)
		if err != nil {
			return false
		}
		return sortedOld == sortedNew
	}
	return strings.TrimSpace(oldValue) == strings.TrimSpace(newValue)
}

func isAccountCapabilitiesKey(key string) bool {
	if !strings.HasPrefix(key, "accounts.") || strings.HasSuffix(key, ".enabled") {
		return false
	}
	return strings.Count(key, ".") == 1
}

func normalizeAccountCapabilities(value string) string {
	capabilities := strings.Split(value, ",")
	normalized := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		capability = strings.TrimSpace(capability)
		if capability == "" {
			continue
		}
		normalized = append(normalized, capability)
	}
	sort.Strings(normalized)
	return strings.Join(normalized, ",")
}

func ToDataSourceConfigMapTFModel(ctx context.Context, diagnostics *diag.Diagnostics, data *structpb.Struct, oldCM tftypes.Map) tftypes.Map {
	if data == nil || len(data.AsMap()) == 0 {
		return ToFilteredConfigMapTFModel(ctx, diagnostics, data, oldCM)
	}
	if oldCM.IsUnknown() || oldCM.IsNull() || len(oldCM.Elements()) == 0 {
		return ToConfigMapTFModel(ctx, diagnostics, data, oldCM)
	}
	return ToFilteredConfigMapTFModel(ctx, diagnostics, data, oldCM)
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

func yamlIsSubset(sub, super any) bool {
	subStr, ok := toYAMLString(sub)
	if !ok {
		return false
	}
	superStr, ok := toYAMLString(super)
	if !ok {
		return false
	}

	var subMap, superMap map[string]any
	if err := yaml.Unmarshal([]byte(subStr), &subMap); err != nil {
		return false
	}
	if err := yaml.Unmarshal([]byte(superStr), &superMap); err != nil {
		return false
	}

	for k, subVal := range subMap {
		superVal, exists := superMap[k]
		if !exists {
			return false
		}
		if !reflect.DeepEqual(subVal, superVal) {
			return false
		}
	}
	return true
}

func toYAMLString(val any) (string, bool) {
	switch v := val.(type) {
	case string:
		return v, true
	case tftypes.String:
		if v.IsNull() || v.IsUnknown() {
			return "", false
		}
		return v.ValueString(), true
	default:
		return "", false
	}
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

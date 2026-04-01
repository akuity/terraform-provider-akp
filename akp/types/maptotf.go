package types

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"sigs.k8s.io/yaml"
)

// ReverseFieldOverride customizes how a specific API map value is converted back to a TF value.
// The tfsdk tag name (snake_case) is used as the key.
//   - mapValue: the value from the API response map (may be nil if the key was missing)
//   - planValue: the current plan/state value for this field (for plan-aware fields)
//
// Return (val, true) to use val; (nil, false) to decline and fall through to default conversion.
type ReverseFieldOverride func(mapValue any, planValue reflect.Value) (attr.Value, bool)

// reverseOverrideMap is a map of reverse field overrides keyed by tfsdk tag name.
type reverseOverrideMap map[string]ReverseFieldOverride

// resolveReverseKey maps a tfsdk snake_case tag to the camelCase API key,
// respecting rename overrides.
func resolveReverseKey(tag string, renames renameMap) string {
	if renames != nil {
		if newKey, ok := renames[tag]; ok {
			return newKey
		}
	}
	return SnakeToCamel(tag)
}

// isNullOrUnknown checks if a TF framework value is null or unknown.
func isNullOrUnknown(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	iface := v.Interface()
	switch val := iface.(type) {
	case types.String:
		return val.IsNull() || val.IsUnknown()
	case types.Bool:
		return val.IsNull() || val.IsUnknown()
	case types.Int64:
		return val.IsNull() || val.IsUnknown()
	case types.Float64:
		return val.IsNull() || val.IsUnknown()
	case types.Object:
		return val.IsNull() || val.IsUnknown()
	case types.List:
		return val.IsNull() || val.IsUnknown()
	case types.Set:
		return val.IsNull() || val.IsUnknown()
	case types.Map:
		return val.IsNull() || val.IsUnknown()
	}

	// For pointer types, nil means "not set"
	if v.Kind() == reflect.Ptr {
		return v.IsNil()
	}
	// For slice types, nil means "not set"
	if v.Kind() == reflect.Slice {
		return v.IsNil()
	}
	return false
}

func isTFFrameworkType(v reflect.Value) bool {
	switch v.Interface().(type) {
	case types.String, types.Bool, types.Int64, types.Float64,
		types.Object, types.List, types.Set, types.Map:
		return true
	}
	return false
}

// setZeroTFValue sets a TF field to its typed zero/empty value.
func setZeroTFValue(fieldVal reflect.Value, fullPath string) {
	switch fieldVal.Interface().(type) {
	case types.String:
		fieldVal.Set(reflect.ValueOf(types.StringValue("")))
	case types.Bool:
		fieldVal.Set(reflect.ValueOf(types.BoolValue(false)))
	case types.Int64:
		fieldVal.Set(reflect.ValueOf(types.Int64Value(0)))
	case types.Float64:
		fieldVal.Set(reflect.ValueOf(types.Float64Value(0)))
	case types.Object:
		// Use registered attr types if available; otherwise nil.
		var attrTypes map[string]attr.Type
		if RegisteredObjectAttrTypes != nil {
			attrTypes = RegisteredObjectAttrTypes[fullPath]
		}
		fieldVal.Set(reflect.ValueOf(types.ObjectNull(attrTypes)))
	case types.List:
		fieldVal.Set(reflect.ValueOf(types.ListNull(types.StringType)))
	case types.Set:
		fieldVal.Set(reflect.ValueOf(types.SetNull(types.StringType)))
	case types.Map:
		mapElemType := attr.Type(types.StringType)
		if attrTypes := lookupAttrTypes(fullPath); attrTypes != nil {
			mapElemType = types.ObjectType{AttrTypes: attrTypes}
		}
		fieldVal.Set(reflect.ValueOf(types.MapNull(mapElemType)))
	default:
		// For pointer types, set to nil; for slices, set to nil slice
		if fieldVal.Kind() == reflect.Ptr {
			fieldVal.Set(reflect.Zero(fieldVal.Type()))
		} else if fieldVal.Kind() == reflect.Slice {
			fieldVal.Set(reflect.Zero(fieldVal.Type()))
		}
	}
}

// toTFString converts an API value to types.String.
func toTFString(v any) types.String {
	if v == nil {
		return types.StringValue("")
	}
	switch val := v.(type) {
	case string:
		return types.StringValue(val)
	case float64:
		// JSON numbers — could be an int represented as float
		if val == float64(int64(val)) {
			return types.StringValue(fmt.Sprintf("%d", int64(val)))
		}
		return types.StringValue(fmt.Sprintf("%g", val))
	case bool:
		return types.StringValue(fmt.Sprintf("%t", val))
	default:
		return types.StringValue(fmt.Sprintf("%v", val))
	}
}

// toTFBool converts an API value to types.Bool.
func toTFBool(v any) types.Bool {
	if v == nil {
		return types.BoolValue(false)
	}
	switch val := v.(type) {
	case bool:
		return types.BoolValue(val)
	case float64:
		return types.BoolValue(val != 0)
	default:
		return types.BoolValue(false)
	}
}

// toTFInt64 converts an API value to types.Int64.
func toTFInt64(v any) types.Int64 {
	if v == nil {
		return types.Int64Value(0)
	}
	switch val := v.(type) {
	case float64:
		return types.Int64Value(int64(val))
	case int64:
		return types.Int64Value(val)
	case int:
		return types.Int64Value(int64(val))
	default:
		return types.Int64Value(0)
	}
}

// toTFFloat64 converts an API value to types.Float64.
func toTFFloat64(v any) types.Float64 {
	if v == nil {
		return types.Float64Value(0)
	}
	switch val := v.(type) {
	case float64:
		return types.Float64Value(val)
	case int64:
		return types.Float64Value(float64(val))
	default:
		return types.Float64Value(0)
	}
}

// inferAttrTypesFromMap infers Terraform attribute types from API map values.
// This is used as a fallback when the plan's types.Object has no attribute type information
// (e.g., zero-value types.Object from an unset computed field).
// JSON numbers are mapped to Float64Type since encoding/json decodes all numbers as float64.
func inferAttrTypesFromMap(m map[string]any) map[string]attr.Type {
	if len(m) == 0 {
		return nil
	}
	result := make(map[string]attr.Type, len(m))
	for k, v := range m {
		snakeKey := CamelToSnake(k)
		result[snakeKey] = inferAttrType(v)
	}
	return result
}

// inferAttrType infers a Terraform attr.Type from a Go value.
func inferAttrType(v any) attr.Type {
	switch val := v.(type) {
	case string:
		return types.StringType
	case bool:
		return types.BoolType
	case float64:
		return types.Float64Type
	case map[string]any:
		nestedTypes := make(map[string]attr.Type, len(val))
		for k, nested := range val {
			snakeKey := CamelToSnake(k)
			nestedTypes[snakeKey] = inferAttrType(nested)
		}
		return types.ObjectType{AttrTypes: nestedTypes}
	case []any:
		if len(val) > 0 {
			return types.ListType{ElemType: inferAttrType(val[0])}
		}
		// Empty list — default to string elements
		return types.ListType{ElemType: types.StringType}
	case nil:
		// Default to string for nil values
		return types.StringType
	default:
		return types.StringType
	}
}

// nullForType returns the typed null value for a given attr.Type.
func zeroForType(t attr.Type) attr.Value {
	switch t {
	case types.StringType:
		return types.StringValue("")
	case types.BoolType:
		return types.BoolValue(false)
	case types.Int64Type:
		return types.Int64Value(0)
	case types.Float64Type:
		return types.Float64Value(0)
	}
	return nullForType(t)
}

func nullForType(t attr.Type) attr.Value {
	switch t {
	case types.StringType:
		return types.StringNull()
	case types.BoolType:
		return types.BoolNull()
	case types.Int64Type:
		return types.Int64Null()
	case types.Float64Type:
		return types.Float64Null()
	}

	switch ct := t.(type) {
	case types.ObjectType:
		return types.ObjectNull(ct.AttrTypes)
	case types.ListType:
		return types.ListNull(ct.ElemType)
	case types.SetType:
		return types.SetNull(ct.ElemType)
	case types.MapType:
		return types.MapNull(ct.ElemType)
	}

	return types.StringNull()
}

func CamelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// --- Reverse Override Helpers ---

// PreserveFromPlan returns an override that always preserves the plan/state value.
// Use for write-only / secret fields that the API may not return (e.g., metrics_ingress_password_hash).
// If the API does return a value, it is used. Otherwise, the plan value is preserved as-is.
func PreserveFromPlan() ReverseFieldOverride {
	return func(mapValue any, planValue reflect.Value) (attr.Value, bool) {
		if mapValue != nil {
			return nil, false
		}
		if planValue.IsValid() && !isNullOrUnknown(planValue) {
			if av, ok := planValue.Interface().(attr.Value); ok {
				return av, true
			}
		}
		return nil, false
	}
}

func HydrateFromAPIWhenPlanNull() ReverseFieldOverride {
	return func(mapValue any, planValue reflect.Value) (attr.Value, bool) {
		if mapValue == nil || !planValue.IsValid() || !planIsNull(planValue) {
			return nil, false
		}

		switch planValue.Interface().(type) {
		case types.Bool:
			return toTFBool(mapValue), true
		case types.String:
			return toTFString(mapValue), true
		case types.Int64:
			return toTFInt64(mapValue), true
		case types.Float64:
			return toTFFloat64(mapValue), true
		}

		return nil, false
	}
}

// TFOnlyField returns an override that always preserves the plan/state value
// and falls back to a default attr.Value when the plan has no value.
// Use for fields that exist only in the TF schema with no API equivalent
// (e.g., remove_agent_resources_on_destroy, reapply_manifests_on_update, kubeconfig).
func TFOnlyField(defaultVal attr.Value) ReverseFieldOverride {
	return func(_ any, planValue reflect.Value) (attr.Value, bool) {
		if planValue.IsValid() && !isNullOrUnknown(planValue) {
			if av, ok := planValue.Interface().(attr.Value); ok {
				return av, true
			}
		}
		return defaultVal, true
	}
}

// ObjectToYAMLString returns an override that converts a map[string]any (from the API)
// to a types.String containing YAML. This is the reverse of yamlStringToObject().
// Use for kustomization fields that are stored as YAML strings in TF but as objects in the API.
func ObjectToYAMLString() ReverseFieldOverride {
	return func(mapValue any, planValue reflect.Value) (attr.Value, bool) {
		if mapValue == nil {
			return types.StringNull(), true
		}
		// Marshal the map to JSON, then convert to YAML
		jsonBytes, err := json.Marshal(mapValue)
		if err != nil {
			return types.StringNull(), true
		}
		var objMap map[string]any
		if err := json.Unmarshal(jsonBytes, &objMap); err != nil {
			return types.StringNull(), true
		}
		if len(objMap) == 0 {
			if planValue.IsValid() && !isNullOrUnknown(planValue) {
				if pv, ok := planValue.Interface().(types.String); ok && pv.ValueString() == "" {
					return types.StringValue(""), true
				}
			}
			return types.StringNull(), true
		}
		yamlBytes, err := yaml.JSONToYAML(jsonBytes)
		if err != nil {
			return types.StringNull(), true
		}
		yamlStr := string(yamlBytes)
		// If we have a plan value, check if the normalized YAML matches — preserve plan if so
		if planValue.IsValid() && !isNullOrUnknown(planValue) {
			if pv, ok := planValue.Interface().(types.String); ok && !pv.IsNull() && !pv.IsUnknown() {
				if normalizedYAMLEqual(pv.ValueString(), yamlStr) {
					return pv, true
				}
			}
		}
		return types.StringValue(yamlStr), true
	}
}

// ValueObjectToMapString returns an override that converts a map of {key: {value: string}}
// objects back to a types.Map of strings. This is the reverse of mapStringToValueObject().
// Use for fields like dex_config_secret where the API wraps string values in {value: "..."}.
func ValueObjectToMapString() ReverseFieldOverride {
	return func(mapValue any, planValue reflect.Value) (attr.Value, bool) {
		if mapValue == nil {
			// API didn't return — preserve plan value (secrets are often not returned)
			if planValue.IsValid() && !isNullOrUnknown(planValue) {
				if av, ok := planValue.Interface().(attr.Value); ok {
					return av, true
				}
			}
			return types.MapNull(types.StringType), true
		}
		objMap, ok := mapValue.(map[string]any)
		if !ok {
			return types.MapNull(types.StringType), true
		}
		elements := make(map[string]attr.Value)
		for k, v := range objMap {
			inner, ok := v.(map[string]any)
			if !ok {
				continue
			}
			if val, ok := inner["value"]; ok {
				elements[k] = toTFString(val)
			}
		}
		if len(elements) == 0 {
			return types.MapNull(types.StringType), true
		}
		m, _ := types.MapValue(types.StringType, elements)
		return m, true
	}
}

// ExcludeFromAPI returns an override that ignores the API value entirely
// and always preserves the plan/state value. Use when the API returns a value
// but the TF field should not be updated from it (e.g., computed fields that
// are derived differently in TF).
func ExcludeFromAPI() ReverseFieldOverride {
	return func(_ any, planValue reflect.Value) (attr.Value, bool) {
		if planValue.IsValid() && !isNullOrUnknown(planValue) {
			if av, ok := planValue.Interface().(attr.Value); ok {
				return av, true
			}
		}
		return nil, true // Signal handled but no value — caller should keep zero
	}
}

// normalizedYAMLEqual compares two YAML strings by normalizing them through
// unmarshal/marshal to handle formatting differences.
func normalizedYAMLEqual(a, b string) bool {
	var aData, bData map[string]any
	if err := yaml.Unmarshal([]byte(a), &aData); err != nil {
		return false
	}
	if err := yaml.Unmarshal([]byte(b), &bData); err != nil {
		return false
	}
	aNorm, err1 := yaml.Marshal(aData)
	bNorm, err2 := yaml.Marshal(bData)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(aNorm) == string(bNorm)
}

func ProtoEnumToLowerString(mapping map[string]string) ReverseFieldOverride {
	return func(mapValue any, _ reflect.Value) (attr.Value, bool) {
		if mapValue == nil {
			return nil, false
		}
		str, ok := mapValue.(string)
		if !ok {
			return nil, false
		}
		if mapped, exists := mapping[str]; exists {
			return types.StringValue(mapped), true
		}
		return nil, false
	}
}

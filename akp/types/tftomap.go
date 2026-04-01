package types

import (
	"encoding/json"
	"reflect"
	"strings"
	"unicode"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"sigs.k8s.io/yaml"
)

// FieldOverride customizes how a specific field is converted.
// The tfsdk tag name (snake_case) is used as the key.
// Return (val, true) to use val; (excludeFieldSentinel, true) to exclude the field;
// (nil, false) to decline the override and fall through to default conversion.
type FieldOverride func(fieldVal reflect.Value) (any, bool)

// excludeFieldSentinelType is a type for the sentinel value to allow it to be a constant.
type excludeFieldSentinelType string

// excludeFieldSentinel is a sentinel value returned by ExcludeField() to signal
// that a field should be unconditionally excluded from the output map.
const excludeFieldSentinel excludeFieldSentinelType = "EXCLUDE_FIELD"

// overrideMap is a map of field overrides with custom logics
type overrideMap map[string]FieldOverride

// renameMap is to rename some fields without following convensions, this is due to some historical reasonst
type renameMap map[string]string

func TFToMapWithOverrides(v any, overrides overrideMap, renames renameMap) map[string]any {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	result := make(map[string]any)
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		tag := field.Tag.Get("tfsdk")
		if tag == "" || tag == "-" {
			continue
		}
		camelKey := resolveKey(tag, renames)

		if overrides != nil {
			if override, exists := overrides[tag]; exists {
				converted, ok := override(fieldVal)
				if ok {
					if converted != excludeFieldSentinel {
						result[camelKey] = converted
					}
				} else {
					converted, ok := convertField(fieldVal)
					if ok {
						result[camelKey] = converted
					}
				}
				continue
			}
		}

		nestedOverrides := extractNestedMap(overrides, tag)
		nestedRenames := extractNestedMap(renames, tag)
		if len(nestedOverrides) > 0 || len(nestedRenames) > 0 {
			converted, ok := convertFieldWithOverrides(fieldVal, nestedOverrides, nestedRenames)
			if ok {
				result[camelKey] = converted
			}
			continue
		}

		converted, ok := convertField(fieldVal)
		if !ok {
			continue
		}
		result[camelKey] = converted
	}

	return result
}

func yamlStringToObject() FieldOverride {
	return func(fieldVal reflect.Value) (any, bool) {
		v, ok := fieldVal.Interface().(types.String)
		if !ok || v.IsNull() || v.IsUnknown() {
			return nil, false
		}
		if v.ValueString() == "" {
			return map[string]any{}, true
		}
		var obj map[string]any
		jsonBytes, err := yaml.YAMLToJSON([]byte(v.ValueString()))
		if err != nil {
			return nil, false
		}
		if err := json.Unmarshal(jsonBytes, &obj); err != nil {
			return nil, false
		}
		if len(obj) == 0 {
			return nil, false
		}
		return obj, true
	}
}

func emptyListIfSet() FieldOverride {
	return func(fieldVal reflect.Value) (any, bool) {
		iface := fieldVal.Interface()
		switch v := iface.(type) {
		case types.List:
			if v.IsNull() || v.IsUnknown() {
				return nil, false
			}
			if len(v.Elements()) == 0 {
				return []any{}, true
			}
			return nil, false
		default:
			rv := reflect.ValueOf(iface)
			if rv.Kind() == reflect.Ptr {
				if rv.IsNil() {
					return nil, false
				}
				rv = rv.Elem()
			}
			if rv.Kind() == reflect.Slice {
				if rv.IsNil() {
					return nil, false
				}
				if rv.Len() == 0 {
					return []any{}, true
				}
			}
			return nil, false
		}
	}
}

func alwaysIncludeString() FieldOverride {
	return func(fieldVal reflect.Value) (any, bool) {
		v, ok := fieldVal.Interface().(types.String)
		if !ok || v.IsNull() || v.IsUnknown() {
			return "", true
		}
		return v.ValueString(), true
	}
}

func mapStringToValueObject() FieldOverride {
	return func(fieldVal reflect.Value) (any, bool) {
		v, ok := fieldVal.Interface().(types.Map)
		if !ok || v.IsNull() || v.IsUnknown() {
			return nil, false
		}
		result := make(map[string]any)
		for k, elem := range v.Elements() {
			strVal, ok := elem.(types.String)
			if !ok || strVal.IsNull() || strVal.IsUnknown() {
				continue
			}
			result[k] = map[string]any{"value": strVal.ValueString()}
		}
		if len(result) == 0 {
			return nil, false
		}
		return result, true
	}
}

func stringWithMapping(mapping map[string]string) FieldOverride {
	return func(fieldVal reflect.Value) (any, bool) {
		v, ok := fieldVal.Interface().(types.String)
		if !ok || v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
			return nil, false
		}
		s := v.ValueString()
		if mapped, ok := mapping[s]; ok {
			return mapped, true
		}
		return s, true
	}
}

func excludeField() FieldOverride {
	return func(fieldVal reflect.Value) (any, bool) {
		return excludeFieldSentinel, true
	}
}

func suppressEmptyString() FieldOverride {
	return func(fieldVal reflect.Value) (any, bool) {
		v, ok := fieldVal.Interface().(types.String)
		if !ok || v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
			return nil, false
		}
		return v.ValueString(), true
	}
}

func extractNestedMap[M ~map[string]V, V any](m M, prefix string) M {
	if m == nil {
		return nil
	}
	var nested M
	pfx := prefix + "."
	for k, v := range m {
		if strings.HasPrefix(k, pfx) {
			if nested == nil {
				nested = make(M)
			}
			nested[strings.TrimPrefix(k, pfx)] = v
		}
	}
	return nested
}

func resolveKey(tag string, renames renameMap) string {
	if renames != nil {
		if newKey, ok := renames[tag]; ok {
			return newKey
		}
	}
	return SnakeToCamel(tag)
}

func convertFieldWithOverrides(fieldVal reflect.Value, overrides overrideMap, renames renameMap) (any, bool) {
	iface := fieldVal.Interface()
	if obj, ok := iface.(types.Object); ok {
		if obj.IsNull() || obj.IsUnknown() {
			return nil, false
		}
		result := make(map[string]any)
		for name, val := range obj.Attributes() {
			camelKey := resolveKey(name, renames)
			if override, exists := overrides[name]; exists {
				attrFieldVal := reflect.ValueOf(val)
				converted, ok := override(attrFieldVal)
				if ok {
					if converted != excludeFieldSentinel {
						result[camelKey] = converted
					}
				} else {
					converted, ok := convertAttrValue(val)
					if ok {
						result[camelKey] = converted
					}
				}
				continue
			}
			nestedOverrides := extractNestedMap(overrides, name)
			nestedRenames := extractNestedMap(renames, name)
			if len(nestedOverrides) > 0 || len(nestedRenames) > 0 {
				converted, ok := convertFieldWithOverrides(reflect.ValueOf(val), nestedOverrides, nestedRenames)
				if ok {
					result[camelKey] = converted
				}
				continue
			}
			converted, ok := convertAttrValue(val)
			if !ok {
				continue
			}
			result[camelKey] = converted
		}
		return result, true
	}

	v := reflect.ValueOf(iface)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		m := TFToMapWithOverrides(v.Interface(), overrides, renames)
		if m == nil {
			return nil, false
		}
		return m, true
	}

	return convertField(fieldVal)
}

func convertField(fieldVal reflect.Value) (any, bool) {
	iface := fieldVal.Interface()

	switch v := iface.(type) {
	case types.String:
		if v.IsNull() || v.IsUnknown() {
			return nil, false
		}
		return v.ValueString(), true
	case types.Bool:
		if v.IsNull() || v.IsUnknown() {
			return nil, false
		}
		return v.ValueBool(), true
	case types.Int64:
		if v.IsNull() || v.IsUnknown() {
			return nil, false
		}
		return v.ValueInt64(), true
	case types.Float64:
		if v.IsNull() || v.IsUnknown() {
			return nil, false
		}
		return v.ValueFloat64(), true
	case types.Object:
		return convertObject(v)
	case types.List:
		return convertList(v)
	case types.Set:
		return convertSet(v)
	case types.Map:
		return convertMap(v)
	default:
		return convertReflect(fieldVal)
	}
}

func convertObject(obj types.Object) (any, bool) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, false
	}
	result := make(map[string]any)
	for name, val := range obj.Attributes() {
		converted, ok := convertAttrValue(val)
		if !ok {
			continue
		}
		result[SnakeToCamel(name)] = converted
	}
	return result, true
}

func convertList(list types.List) (any, bool) {
	if list.IsNull() || list.IsUnknown() {
		return nil, false
	}
	result := make([]any, 0)
	for _, elem := range list.Elements() {
		converted, ok := convertAttrValue(elem)
		if ok {
			result = append(result, converted)
		}
	}
	return result, true
}

func convertSet(set types.Set) (any, bool) {
	if set.IsNull() || set.IsUnknown() {
		return nil, false
	}
	result := make([]any, 0)
	for _, elem := range set.Elements() {
		converted, ok := convertAttrValue(elem)
		if ok {
			result = append(result, converted)
		}
	}
	return result, true
}

func convertMap(m types.Map) (any, bool) {
	if m.IsNull() || m.IsUnknown() {
		return nil, false
	}
	result := make(map[string]any)
	for k, v := range m.Elements() {
		converted, ok := convertAttrValue(v)
		if ok {
			result[k] = converted
		}
	}
	return result, true
}

func convertAttrValue(val attr.Value) (any, bool) {
	switch v := val.(type) {
	case types.String:
		if v.IsNull() || v.IsUnknown() {
			return nil, false
		}
		return v.ValueString(), true
	case types.Bool:
		if v.IsNull() || v.IsUnknown() {
			return nil, false
		}
		return v.ValueBool(), true
	case types.Int64:
		if v.IsNull() || v.IsUnknown() {
			return nil, false
		}
		return v.ValueInt64(), true
	case types.Float64:
		if v.IsNull() || v.IsUnknown() {
			return nil, false
		}
		return v.ValueFloat64(), true
	case types.Object:
		return convertObject(v)
	case types.List:
		return convertList(v)
	case types.Set:
		return convertSet(v)
	case types.Map:
		return convertMap(v)
	default:
		return nil, false
	}
}

func convertReflect(v reflect.Value) (any, bool) {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return nil, false
		}
		return convertReflect(v.Elem())
	case reflect.Struct:
		m := TFToMapWithOverrides(v.Interface(), nil, nil)
		if m == nil {
			return nil, false
		}
		return m, true
	case reflect.Slice:
		if v.IsNil() {
			return nil, false
		}
		result := make([]any, 0)
		for i := 0; i < v.Len(); i++ {
			converted, ok := convertField(v.Index(i))
			if ok {
				result = append(result, converted)
			}
		}
		return result, true
	default:
		return nil, false
	}
}

func SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			runes := []rune(parts[i])
			runes[0] = unicode.ToUpper(runes[0])
			parts[i] = string(runes)
		}
	}
	return strings.Join(parts, "")
}

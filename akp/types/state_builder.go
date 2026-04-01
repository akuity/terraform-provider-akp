package types

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type readContextKey struct{}

func WithReadContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, readContextKey{}, true)
}

func isReadContext(ctx context.Context) bool {
	v, _ := ctx.Value(readContextKey{}).(bool)
	return v
}

func WithoutReadContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, readContextKey{}, false)
}

// BuildStateFromAPI constructs a TF state struct from an API response map, guided by the
// plan/prior-state and registered schema metadata. Unlike MapToTFWithOverrides which patches
// the existing struct in place, this function builds the state from first principles using
// clear rules:
//
//  1. If the field has a modifier that explicitly preserves null config/state
//     (UseStateForNullUnknown/IgnoreWhenNotConfigured) AND the plan value is null/nil
//     → state = null (plan wins, regardless of what API returns)
//  2. If the API has a value → convert and use it
//  3. If the API doesn't have a value → preserve plan value; if plan is also empty, use zero
//
// Parameters:
//   - ctx: context for diagnostics
//   - apiMap: the API response as map[string]any (from structpb.Struct.AsMap())
//   - tfStruct: pointer to the TF struct to populate (output)
//   - planStruct: pointer to the plan/state TF struct (for plan-aware decisions); may be nil
//   - overrides: custom field conversion logic (same as MapToTFWithOverrides)
//   - renames: API key overrides (same as MapToTFWithOverrides)
//   - pathPrefix: dot-separated prefix for schema registry lookups (e.g., "argocd")
func BuildStateFromAPI(
	ctx context.Context,
	apiMap map[string]any,
	tfStruct any,
	planStruct any,
	overrides reverseOverrideMap,
	renames renameMap,
	pathPrefix string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	tfVal := reflect.ValueOf(tfStruct)
	if tfVal.Kind() != reflect.Ptr || tfVal.IsNil() {
		diags.AddError("BuildStateFromAPI", "tfStruct must be a non-nil pointer to a struct")
		return diags
	}
	tfVal = tfVal.Elem()
	if tfVal.Kind() != reflect.Struct {
		diags.AddError("BuildStateFromAPI", "tfStruct must point to a struct")
		return diags
	}

	var planVal reflect.Value
	if planStruct != nil {
		planVal = reflect.ValueOf(planStruct)
		if planVal.Kind() == reflect.Ptr {
			planVal = planVal.Elem()
		}
	}

	diags.Append(buildStruct(ctx, apiMap, tfVal, planVal, overrides, renames, pathPrefix)...)
	return diags
}

// buildStruct walks a single TF struct level and populates each field from the API map.
func buildStruct(
	ctx context.Context,
	apiMap map[string]any,
	tfVal reflect.Value,
	planVal reflect.Value,
	overrides reverseOverrideMap,
	renames renameMap,
	pathPrefix string,
) diag.Diagnostics {
	var diags diag.Diagnostics
	typ := tfVal.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := tfVal.Field(i)

		tag := field.Tag.Get("tfsdk")
		if tag == "" || tag == "-" {
			continue
		}

		// Build full dot-separated path for registry lookups
		fullPath := tag
		if pathPrefix != "" {
			fullPath = pathPrefix + "." + tag
		}

		// Get the plan field value for this field
		var planFieldVal reflect.Value
		if planVal.IsValid() && planVal.Kind() == reflect.Struct {
			planFieldVal = planVal.FieldByName(field.Name)
		}

		// --- Rule 0: Check overrides first (handles special cases) ---
		if overrides != nil {
			if override, ok := overrides[tag]; ok {
				apiKey := resolveReverseKey(tag, renames)
				mapValue := apiMap[apiKey]
				if val, handled := override(mapValue, planFieldVal); handled {
					if val == nil {
						setZeroTFValue(fieldVal, fullPath)
					} else if isReadContext(ctx) && val.IsNull() && !isSensitiveField(fullPath) {
						setZeroTFValue(fieldVal, fullPath)
					} else {
						fieldVal.Set(reflect.ValueOf(val))
					}
					continue
				}
			}
		}
		// --- Rule 0b: Sensitive fields — always preserve plan value ---
		if planFieldVal.IsValid() && isSensitiveField(fullPath) {
			if !isNullOrUnknown(planFieldVal) {
				fieldVal.Set(planFieldVal)
			}
			continue
		}

		nestedOverrides := extractNestedMap(overrides, tag)
		nestedRenames := extractNestedMap(renames, tag)
		apiKey := resolveReverseKey(tag, renames)
		mapValue, exists := apiMap[apiKey]

		// --- Import path: populate from API, no plan-preservation logic ---
		if isReadContext(ctx) {
			if !exists || mapValue == nil {
				setZeroTFValue(fieldVal, fullPath)
			} else if field.Type.Kind() == reflect.Ptr && isAllProtobufDefaults(mapValue) {
				// Pointer fields with only default values: keep nil to match apply behavior
			} else {
				diags.Append(setFieldFromAPI(ctx, fieldVal, mapValue, planFieldVal, nestedOverrides, nestedRenames, fullPath)...)
			}
			continue
		}

		// --- Apply path below (Create/Update/normal Read) ---

		// --- Rule 1a: Plan-wins for null/nil fields with explicit null-preserving modifiers ---
		if planFieldVal.IsValid() && planIsNull(planFieldVal) && shouldPreserveNullFromPlan(fullPath) {
			preserveNullField(ctx, fieldVal, field.Type, fullPath)
			continue
		}

		// --- Rule 1b: Preserve null for non-computed fields by default.
		// Targeted reverse overrides can opt specific fields into import/read hydration
		// before this branch runs (for example spec.namespace_scoped).
		if planFieldVal.IsValid() && planIsNull(planFieldVal) {
			if !isComputedField(fullPath) {
				preserveNullField(ctx, fieldVal, field.Type, fullPath)
				continue
			}
			// Computed pointer fields: skip if API returned empty map (preserve null)
			if field.Type.Kind() == reflect.Ptr && isEmptyAPIMap(mapValue) {
				continue
			}
		}

		// --- Rule 2: API has no value for this field ---
		if !exists || mapValue == nil {
			if planFieldVal.IsValid() && !isNullOrUnknown(planFieldVal) {
				fieldVal.Set(planFieldVal)
			} else if planFieldVal.IsValid() && planIsNull(planFieldVal) {
				preserveNullField(ctx, fieldVal, field.Type, fullPath)
			} else if !planVal.IsValid() {
				setZeroTFValue(fieldVal, fullPath)
			} else {
				preserveNullField(ctx, fieldVal, field.Type, fullPath)
			}
			continue
		}

		// --- Rule 3: API has a value — convert it ---
		diags.Append(setFieldFromAPI(ctx, fieldVal, mapValue, planFieldVal, nestedOverrides, nestedRenames, fullPath)...)
	}
	return diags
}

// planIsNull returns true if the plan value represents null/nil.
// IMPORTANT: Unknown is NOT null. Unknown means "server decides" and the API value should be used.
func planIsNull(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	// Check TF framework types — only IsNull(), NOT IsUnknown()
	switch val := v.Interface().(type) {
	case types.Object:
		return val.IsNull()
	case types.String:
		return val.IsNull()
	case types.Bool:
		return val.IsNull()
	case types.Int64:
		return val.IsNull()
	case types.Float64:
		return val.IsNull()
	case types.List:
		return val.IsNull()
	case types.Map:
		return val.IsNull()
	case types.Set:
		return val.IsNull()
	}
	// Pointer types: nil = null
	if v.Kind() == reflect.Ptr {
		return v.IsNil()
	}
	// Slice types: nil = null
	if v.Kind() == reflect.Slice {
		return v.IsNil()
	}
	return false
}

// shouldPreserveNullFromPlan returns true if the field's plan modifier explicitly
// says "preserve null config/state".
func shouldPreserveNullFromPlan(fullPath string) bool {
	if RegisteredPlanModifiers == nil {
		return false // No metadata — cannot make metadata-driven decisions
	}
	mod := RegisteredPlanModifiers[fullPath]
	return mod == PlanModUseStateForNullUnknown || mod == PlanModIgnoreWhenNotConfigured
}

func isSensitiveField(fullPath string) bool {
	return RegisteredSensitiveFields[fullPath]
}

func isComputedField(fullPath string) bool {
	return RegisteredComputedFields[fullPath]
}

// preserveNullField sets the field to its null representation based on type.
func preserveNullField(ctx context.Context, fieldVal reflect.Value, fieldType reflect.Type, fullPath string) {
	switch {
	case fieldType.Kind() == reflect.Ptr:
		// nil pointer = TF null for pointer-to-struct fields
		fieldVal.Set(reflect.Zero(fieldType))

	case isTFFrameworkType(fieldVal):
		// Use typed null for types.Object, types.List, etc.
		switch fieldVal.Interface().(type) {
		case types.Object:
			attrTypes := lookupAttrTypes(fullPath)
			fieldVal.Set(reflect.ValueOf(types.ObjectNull(attrTypes)))
		case types.List:
			fieldVal.Set(reflect.ValueOf(types.ListNull(types.StringType)))
		case types.Map:
			mapElemType := attr.Type(types.StringType)
			if attrTypes := lookupAttrTypes(fullPath); attrTypes != nil {
				mapElemType = types.ObjectType{AttrTypes: attrTypes}
			}
			fieldVal.Set(reflect.ValueOf(types.MapNull(mapElemType)))
		case types.Set:
			fieldVal.Set(reflect.ValueOf(types.SetNull(types.StringType)))
		case types.String:
			fieldVal.Set(reflect.ValueOf(types.StringNull()))
		case types.Bool:
			fieldVal.Set(reflect.ValueOf(types.BoolNull()))
		case types.Int64:
			fieldVal.Set(reflect.ValueOf(types.Int64Null()))
		case types.Float64:
			fieldVal.Set(reflect.ValueOf(types.Float64Null()))
		}

	default:
		fieldVal.Set(reflect.Zero(fieldType))
	}
}

// lookupAttrTypes returns the attr.Type map for a types.Object field from the registry.
func lookupAttrTypes(fullPath string) map[string]attr.Type {
	if RegisteredObjectAttrTypes != nil {
		if at, ok := RegisteredObjectAttrTypes[fullPath]; ok {
			return at
		}
	}
	return nil
}

func isAllProtobufDefaults(val any) bool {
	if val == nil {
		return true
	}
	m, ok := val.(map[string]any)
	if !ok {
		return isProtobufDefault(val)
	}
	for _, v := range m {
		if !isAllProtobufDefaults(v) {
			return false
		}
	}
	return true
}

func isEmptyAPIMap(val any) bool {
	if val == nil {
		return true
	}
	m, ok := val.(map[string]any)
	if !ok {
		return false
	}
	return len(m) == 0
}

func isProtobufDefault(val any) bool {
	if val == nil {
		return true
	}
	switch t := val.(type) {
	case string:
		return t == ""
	case bool:
		return !t
	case float64:
		return t == 0
	}
	return false
}

// setFieldFromAPI converts an API map value and sets it on the TF struct field.
func setFieldFromAPI(
	ctx context.Context,
	fieldVal reflect.Value,
	mapValue any,
	planFieldVal reflect.Value,
	nestedOverrides reverseOverrideMap,
	nestedRenames renameMap,
	fullPath string,
) diag.Diagnostics {
	var diags diag.Diagnostics
	fieldType := fieldVal.Type()

	// Handle TF framework types first
	if isTFFrameworkType(fieldVal) {
		diags.Append(setTFFrameworkField(ctx, fieldVal, mapValue, planFieldVal, nestedOverrides, nestedRenames, fullPath)...)
		return diags
	}

	// Handle Go native types
	switch fieldType.Kind() {
	case reflect.Ptr:
		if fieldType.Elem().Kind() == reflect.Struct {
			subMap, ok := mapValue.(map[string]any)
			if !ok {
				return diags
			}
			// API has data — recurse into the struct
			newStruct := reflect.New(fieldType.Elem())
			var planSub reflect.Value
			if planFieldVal.IsValid() && planFieldVal.Kind() == reflect.Ptr && !planFieldVal.IsNil() {
				planSub = planFieldVal.Elem()
			}
			diags.Append(buildStruct(ctx, subMap, newStruct.Elem(), planSub, nestedOverrides, nestedRenames, fullPath)...)
			fieldVal.Set(newStruct)
		}

	case reflect.Struct:
		subMap, ok := mapValue.(map[string]any)
		if !ok {
			return diags
		}
		var planSub reflect.Value
		if planFieldVal.IsValid() {
			planSub = planFieldVal
		}
		diags.Append(buildStruct(ctx, subMap, fieldVal, planSub, nestedOverrides, nestedRenames, fullPath)...)

	case reflect.Slice:
		diags.Append(setSliceFromAPI(ctx, fieldVal, mapValue, planFieldVal, nestedOverrides, nestedRenames, fullPath)...)

	default:
		// Simple Go types (shouldn't happen in TF structs but handle gracefully)
		fieldVal.Set(reflect.ValueOf(mapValue).Convert(fieldType))
	}

	return diags
}

// setTFFrameworkField handles conversion for types.String, types.Bool, types.Object, etc.
func setTFFrameworkField(
	ctx context.Context,
	fieldVal reflect.Value,
	mapValue any,
	planFieldVal reflect.Value,
	nestedOverrides reverseOverrideMap,
	nestedRenames renameMap,
	fullPath string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	switch fieldVal.Interface().(type) {
	case types.String:
		if planFieldVal.IsValid() && planIsNull(planFieldVal) && isProtobufDefault(mapValue) {
			if isComputedField(fullPath) {
				fieldVal.Set(reflect.ValueOf(toTFString(mapValue)))
			} else {
				fieldVal.Set(reflect.ValueOf(types.StringNull()))
			}
		} else if planFieldVal.IsValid() && !isNullOrUnknown(planFieldVal) && isProtobufDefault(mapValue) {
			fieldVal.Set(planFieldVal)
		} else {
			fieldVal.Set(reflect.ValueOf(toTFString(mapValue)))
		}

	case types.Bool:
		if planFieldVal.IsValid() && planIsNull(planFieldVal) && isProtobufDefault(mapValue) {
			if isComputedField(fullPath) {
				fieldVal.Set(reflect.ValueOf(toTFBool(mapValue)))
			} else {
				fieldVal.Set(reflect.ValueOf(types.BoolNull()))
			}
		} else if planFieldVal.IsValid() && !isNullOrUnknown(planFieldVal) && isProtobufDefault(mapValue) {
			fieldVal.Set(planFieldVal)
		} else {
			fieldVal.Set(reflect.ValueOf(toTFBool(mapValue)))
		}

	case types.Int64:
		if planFieldVal.IsValid() && planIsNull(planFieldVal) && isProtobufDefault(mapValue) {
			if isComputedField(fullPath) {
				fieldVal.Set(reflect.ValueOf(toTFInt64(mapValue)))
			} else {
				fieldVal.Set(reflect.ValueOf(types.Int64Null()))
			}
		} else if planFieldVal.IsValid() && !isNullOrUnknown(planFieldVal) && isProtobufDefault(mapValue) {
			fieldVal.Set(planFieldVal)
		} else {
			fieldVal.Set(reflect.ValueOf(toTFInt64(mapValue)))
		}

	case types.Float64:
		if planFieldVal.IsValid() && planIsNull(planFieldVal) && isProtobufDefault(mapValue) {
			if isComputedField(fullPath) {
				fieldVal.Set(reflect.ValueOf(toTFFloat64(mapValue)))
			} else {
				fieldVal.Set(reflect.ValueOf(types.Float64Null()))
			}
		} else if planFieldVal.IsValid() && !isNullOrUnknown(planFieldVal) && isProtobufDefault(mapValue) {
			fieldVal.Set(planFieldVal)
		} else {
			fieldVal.Set(reflect.ValueOf(toTFFloat64(mapValue)))
		}

	case types.Object:
		obj, d := buildTFObject(ctx, mapValue, planFieldVal, nestedOverrides, nestedRenames, fullPath)
		diags.Append(d...)
		if obj != nil {
			fieldVal.Set(reflect.ValueOf(*obj))
		}

	case types.List:
		list, d := buildTFList(ctx, mapValue, planFieldVal, fullPath)
		diags.Append(d...)
		if list != nil {
			fieldVal.Set(reflect.ValueOf(*list))
		}

	case types.Set:
		set, d := buildTFSet(ctx, mapValue, planFieldVal, fullPath)
		diags.Append(d...)
		if set != nil {
			fieldVal.Set(reflect.ValueOf(*set))
		}

	case types.Map:
		m, d := buildTFMap(ctx, mapValue, planFieldVal, fullPath)
		diags.Append(d...)
		if m != nil {
			fieldVal.Set(reflect.ValueOf(*m))
		}
	}

	return diags
}

// buildTFObject converts a map[string]any API value into a types.Object.
func buildTFObject(
	ctx context.Context,
	mapValue any,
	planFieldVal reflect.Value,
	overrides reverseOverrideMap,
	renames renameMap,
	fullPath string,
) (*types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Determine attrTypes from plan or registry
	attrTypes := lookupAttrTypes(fullPath)
	var planAttrs map[string]attr.Value
	if planFieldVal.IsValid() {
		if pv, ok := planFieldVal.Interface().(types.Object); ok {
			if at := pv.AttributeTypes(ctx); len(at) > 0 {
				attrTypes = at
			}
			if !pv.IsNull() && !pv.IsUnknown() {
				planAttrs = pv.Attributes()
			}
		}
	}

	if len(attrTypes) == 0 {
		// Last resort: infer from the map
		if m, ok := mapValue.(map[string]any); ok {
			attrTypes = inferAttrTypesFromMap(m)
		}
	}

	objMap, ok := mapValue.(map[string]any)
	if !ok {
		null := types.ObjectNull(attrTypes)
		return &null, diags
	}

	// Build attributes from the API map
	attrs := make(map[string]attr.Value)
	for name, attrType := range attrTypes {
		apiKey := SnakeToCamel(name)
		if renames != nil {
			if renamed, ok := renames[name]; ok {
				apiKey = renamed
			}
		}

		apiVal, exists := objMap[apiKey]
		if !exists || apiVal == nil {
			// Use plan value or null
			if planAttrs != nil {
				if pv, ok := planAttrs[name]; ok {
					attrs[name] = pv
					continue
				}
			}
			attrs[name] = nullForType(attrType)
			continue
		}

		// Check override
		if overrides != nil {
			nestedOverrides := extractNestedMap(overrides, name)
			if override, ok := overrides[name]; ok {
				var planAttrVal reflect.Value
				if planAttrs != nil {
					if pv, ok := planAttrs[name]; ok {
						planAttrVal = reflect.ValueOf(pv)
					}
				}
				if val, handled := override(apiVal, planAttrVal); handled {
					attrFullPath := fullPath + "." + name
					if val == nil {
						attrs[name] = nullForType(attrType)
					} else if isReadContext(ctx) && val.IsNull() && !isSensitiveField(attrFullPath) {
						attrs[name] = zeroForType(attrType)
					} else {
						attrs[name] = val
					}
					continue
				}
			}
			_ = nestedOverrides // TODO: deep nested overrides for types.Object children
		}

		attrs[name] = convertAPIValueToAttr(ctx, apiVal, attrType, fullPath+"."+name)
	}

	obj, d := types.ObjectValue(attrTypes, attrs)
	diags.Append(d...)
	return &obj, diags
}

// convertAPIValueToAttr converts a single API value to an attr.Value given the expected type.
func convertAPIValueToAttr(ctx context.Context, apiVal any, attrType attr.Type, fullPath string) attr.Value {
	switch attrType {
	case types.StringType:
		return toTFString(apiVal)
	case types.BoolType:
		return toTFBool(apiVal)
	case types.Int64Type:
		return toTFInt64(apiVal)
	case types.Float64Type:
		return toTFFloat64(apiVal)
	}

	switch ct := attrType.(type) {
	case types.ObjectType:
		if m, ok := apiVal.(map[string]any); ok {
			childAttrs := make(map[string]attr.Value)
			for name, childType := range ct.AttrTypes {
				apiKey := SnakeToCamel(name)
				if cv, exists := m[apiKey]; exists && cv != nil {
					childAttrs[name] = convertAPIValueToAttr(ctx, cv, childType, fullPath+"."+name)
				} else {
					childAttrs[name] = nullForType(childType)
				}
			}
			obj, _ := types.ObjectValue(ct.AttrTypes, childAttrs)
			return obj
		}
		return types.ObjectNull(ct.AttrTypes)

	case types.ListType:
		return convertAPIListToAttr(ctx, apiVal, ct.ElemType, fullPath)

	case types.SetType:
		return convertAPISetToAttr(ctx, apiVal, ct.ElemType, fullPath)

	case types.MapType:
		if m, ok := apiVal.(map[string]any); ok {
			elems := make(map[string]attr.Value)
			for k, v := range m {
				elems[k] = convertAPIValueToAttr(ctx, v, ct.ElemType, fullPath+"."+k)
			}
			mv, _ := types.MapValue(ct.ElemType, elems)
			return mv
		}
		return types.MapNull(ct.ElemType)
	}

	return toTFString(apiVal)
}

func convertAPIListToAttr(ctx context.Context, apiVal any, elemType attr.Type, fullPath string) attr.Value {
	slice, ok := apiVal.([]any)
	if !ok {
		return types.ListNull(elemType)
	}
	elems := make([]attr.Value, 0, len(slice))
	for i, item := range slice {
		elems = append(elems, convertAPIValueToAttr(ctx, item, elemType, fmt.Sprintf("%s[%d]", fullPath, i)))
	}
	list, _ := types.ListValue(elemType, elems)
	return list
}

func convertAPISetToAttr(ctx context.Context, apiVal any, elemType attr.Type, fullPath string) attr.Value {
	slice, ok := apiVal.([]any)
	if !ok {
		return types.SetNull(elemType)
	}
	elems := make([]attr.Value, 0, len(slice))
	for i, item := range slice {
		elems = append(elems, convertAPIValueToAttr(ctx, item, elemType, fmt.Sprintf("%s[%d]", fullPath, i)))
	}
	set, _ := types.SetValue(elemType, elems)
	return set
}

// setSliceFromAPI handles conversion for Go slice fields (e.g., []*RepoServerDelegate).
func setSliceFromAPI(
	ctx context.Context,
	fieldVal reflect.Value,
	mapValue any,
	planFieldVal reflect.Value,
	nestedOverrides reverseOverrideMap,
	nestedRenames renameMap,
	fullPath string,
) diag.Diagnostics {
	var diags diag.Diagnostics
	fieldType := fieldVal.Type()
	elemType := fieldType.Elem()

	apiSlice, ok := mapValue.([]any)
	if !ok {
		return diags
	}

	result := reflect.MakeSlice(fieldType, 0, len(apiSlice))
	for i, item := range apiSlice {
		// Extract plan element at matching index (if available)
		var planElem reflect.Value
		if planFieldVal.IsValid() && planFieldVal.Kind() == reflect.Slice && i < planFieldVal.Len() {
			planElem = planFieldVal.Index(i)
		}

		switch {
		case elemType.Kind() == reflect.Ptr && elemType.Elem().Kind() == reflect.Struct:
			subMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			newElem := reflect.New(elemType.Elem())
			var planSub reflect.Value
			if planElem.IsValid() && planElem.Kind() == reflect.Ptr && !planElem.IsNil() {
				planSub = planElem.Elem()
			}
			diags.Append(buildStruct(ctx, subMap, newElem.Elem(), planSub, nestedOverrides, nestedRenames, fullPath)...)
			result = reflect.Append(result, newElem)

		case elemType.Kind() == reflect.Struct && !isTFFrameworkTypeByType(elemType):
			subMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			newElem := reflect.New(elemType).Elem()
			var planSub reflect.Value
			if planElem.IsValid() && planElem.Kind() == reflect.Struct {
				planSub = planElem
			}
			diags.Append(buildStruct(ctx, subMap, newElem, planSub, nestedOverrides, nestedRenames, fullPath)...)
			result = reflect.Append(result, newElem)
		default:
			// TF types in slices (e.g., []types.String)
			result = reflect.Append(result, reflect.ValueOf(convertGoValueToTFType(item, elemType)))
		}
	}

	if planFieldVal.IsValid() && planFieldVal.Kind() == reflect.Slice &&
		planFieldVal.Len() == result.Len() && isTFFrameworkTypeByType(elemType) {
		if sameElements(planFieldVal, result) {
			fieldVal.Set(planFieldVal)
			return diags
		}
	}

	fieldVal.Set(result)
	return diags
}

func sameElements(a, b reflect.Value) bool {
	if a.Len() != b.Len() {
		return false
	}
	used := make([]bool, b.Len())
	for i := 0; i < a.Len(); i++ {
		aStr := fmt.Sprintf("%v", a.Index(i).Interface())
		found := false
		for j := 0; j < b.Len(); j++ {
			if !used[j] && fmt.Sprintf("%v", b.Index(j).Interface()) == aStr {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// convertGoValueToTFType converts a Go value to the corresponding TF type.
func convertGoValueToTFType(val any, targetType reflect.Type) any {
	switch targetType {
	case reflect.TypeOf(types.String{}):
		return toTFString(val)
	case reflect.TypeOf(types.Bool{}):
		return toTFBool(val)
	case reflect.TypeOf(types.Int64{}):
		return toTFInt64(val)
	case reflect.TypeOf(types.Float64{}):
		return toTFFloat64(val)
	}
	return val
}

func isTFFrameworkTypeByType(t reflect.Type) bool {
	switch t {
	case reflect.TypeOf(types.String{}),
		reflect.TypeOf(types.Bool{}),
		reflect.TypeOf(types.Int64{}),
		reflect.TypeOf(types.Float64{}),
		reflect.TypeOf(types.Object{}),
		reflect.TypeOf(types.List{}),
		reflect.TypeOf(types.Set{}),
		reflect.TypeOf(types.Map{}):
		return true
	}
	return false
}

// buildTFList converts an API []any value into a types.List.
func buildTFList(
	ctx context.Context,
	mapValue any,
	planFieldVal reflect.Value,
	fullPath string,
) (*types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Determine element type from plan or fallback
	elemType := attr.Type(types.StringType) // default
	if planFieldVal.IsValid() {
		if pv, ok := planFieldVal.Interface().(types.List); ok {
			elemType = pv.ElementType(ctx)
		}
	}

	apiSlice, ok := mapValue.([]any)
	if !ok {
		null := types.ListNull(elemType)
		return &null, diags
	}

	elems := make([]attr.Value, 0, len(apiSlice))
	for i, item := range apiSlice {
		elems = append(elems, convertAPIValueToAttr(ctx, item, elemType, fmt.Sprintf("%s[%d]", fullPath, i)))
	}

	list, d := types.ListValue(elemType, elems)
	diags.Append(d...)
	return &list, diags
}

// buildTFSet converts an API []any value into a types.Set.
func buildTFSet(
	ctx context.Context,
	mapValue any,
	planFieldVal reflect.Value,
	fullPath string,
) (*types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	elemType := attr.Type(types.StringType)
	if planFieldVal.IsValid() {
		if pv, ok := planFieldVal.Interface().(types.Set); ok {
			elemType = pv.ElementType(ctx)
		}
	}

	apiSlice, ok := mapValue.([]any)
	if !ok {
		null := types.SetNull(elemType)
		return &null, diags
	}

	elems := make([]attr.Value, 0, len(apiSlice))
	for i, item := range apiSlice {
		elems = append(elems, convertAPIValueToAttr(ctx, item, elemType, fmt.Sprintf("%s[%d]", fullPath, i)))
	}

	set, d := types.SetValue(elemType, elems)
	diags.Append(d...)
	return &set, diags
}

// buildTFMap converts an API map[string]any value into a types.Map.
func buildTFMap(
	ctx context.Context,
	mapValue any,
	planFieldVal reflect.Value,
	fullPath string,
) (*types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics

	elemType := attr.Type(types.StringType)
	if planFieldVal.IsValid() {
		if pv, ok := planFieldVal.Interface().(types.Map); ok && !pv.IsNull() && !pv.IsUnknown() {
			elemType = pv.ElementType(ctx)
		}
	}
	if elemType == types.StringType {
		if attrTypes := lookupAttrTypes(fullPath); attrTypes != nil {
			elemType = types.ObjectType{AttrTypes: attrTypes}
		}
	}

	objMap, ok := mapValue.(map[string]any)
	if !ok {
		null := types.MapNull(elemType)
		return &null, diags
	}

	elems := make(map[string]attr.Value)
	for k, v := range objMap {
		elems[k] = convertAPIValueToAttr(ctx, v, elemType, fullPath+"."+k)
	}

	m, d := types.MapValue(elemType, elems)
	diags.Append(d...)
	return &m, diags
}

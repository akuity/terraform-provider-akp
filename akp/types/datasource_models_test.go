//go:build !acc

package types

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	frameworktypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

var (
	tfStringType = reflect.TypeOf(frameworktypes.String{})
	tfBoolType   = reflect.TypeOf(frameworktypes.Bool{})
	tfInt64Type  = reflect.TypeOf(frameworktypes.Int64{})
	tfMapType    = reflect.TypeOf(frameworktypes.Map{})
	tfObjectType = reflect.TypeOf(frameworktypes.Object{})
)

var instanceDataSourceExcludedTags = map[string]struct{}{
	"argocd_secret":                    {},
	"application_set_secret":           {},
	"argocd_notifications_secret":      {},
	"argocd_image_updater_secret":      {},
	"repo_credential_secrets":          {},
	"repo_template_credential_secrets": {},
	"metrics_ingress_password_hash":    {},
}

var kargoDataSourceExcludedTags = map[string]struct{}{
	"kargo_secret":      {},
	"dex_config_secret": {},
}

// If this test fails, a new non-secret field was added to the resource model
// without being added to the data source projection, or a projected field was
// added but not populated by NewInstanceDataSourceModel.
func TestNewInstanceDataSourceModelMatchesResourceModel(t *testing.T) {
	instance := populateProjectionFixture[Instance](t, "instance")
	projected := NewInstanceDataSourceModel(&instance)

	assertProjectionMatches(t, reflect.ValueOf(instance), reflect.ValueOf(projected), instanceDataSourceExcludedTags, "instance", "NewInstanceDataSourceModel")
}

// If this test fails, a new non-secret field was added to the resource model
// without being added to the data source projection, or a projected field was
// added but not populated by NewKargoInstanceDataSourceModel.
func TestNewKargoInstanceDataSourceModelMatchesResourceModel(t *testing.T) {
	instance := populateProjectionFixture[KargoInstance](t, "kargo_instance")
	projected := NewKargoInstanceDataSourceModel(&instance)

	assertProjectionMatches(t, reflect.ValueOf(instance), reflect.ValueOf(projected), kargoDataSourceExcludedTags, "kargo_instance", "NewKargoInstanceDataSourceModel")
}

func populateProjectionFixture[T any](t *testing.T, root string) T {
	t.Helper()

	var zero T
	value := populateProjectionValue(t, reflect.TypeOf(zero), root)
	return value.Interface().(T)
}

func populateProjectionValue(t *testing.T, typ reflect.Type, path string) reflect.Value {
	t.Helper()

	switch typ {
	case tfStringType:
		return reflect.ValueOf(frameworktypes.StringValue(path))
	case tfBoolType:
		return reflect.ValueOf(frameworktypes.BoolValue(true))
	case tfInt64Type:
		return reflect.ValueOf(frameworktypes.Int64Value(42))
	case tfMapType:
		return reflect.ValueOf(frameworktypes.MapValueMust(
			frameworktypes.StringType,
			map[string]attr.Value{"value": frameworktypes.StringValue(path)},
		))
	case tfObjectType:
		return reflect.ValueOf(frameworktypes.ObjectValueMust(
			map[string]attr.Type{"value": frameworktypes.StringType},
			map[string]attr.Value{"value": frameworktypes.StringValue(path)},
		))
	}

	switch typ.Kind() {
	case reflect.Pointer:
		value := reflect.New(typ.Elem())
		value.Elem().Set(populateProjectionValue(t, typ.Elem(), path))
		return value
	case reflect.Struct:
		value := reflect.New(typ).Elem()
		for idx := 0; idx < typ.NumField(); idx++ {
			field := typ.Field(idx)
			if !field.IsExported() || !value.Field(idx).CanSet() {
				continue
			}
			value.Field(idx).Set(populateProjectionValue(t, field.Type, fieldPath(path, field)))
		}
		return value
	case reflect.Slice:
		value := reflect.MakeSlice(typ, 1, 1)
		value.Index(0).Set(populateProjectionValue(t, typ.Elem(), path+"[0]"))
		return value
	case reflect.Map:
		require.Equal(t, reflect.String, typ.Key().Kind(), "unsupported non-string map key for %s", path)
		value := reflect.MakeMapWithSize(typ, 1)
		key := reflect.ValueOf("value").Convert(typ.Key())
		elem := populateProjectionValue(t, typ.Elem(), path+".value")
		value.SetMapIndex(key, elem)
		return value
	case reflect.String:
		return reflect.ValueOf(path).Convert(typ)
	case reflect.Bool:
		return reflect.ValueOf(true).Convert(typ)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.ValueOf(int64(42)).Convert(typ)
	default:
		require.Failf(t, "unsupported projection fixture type", "type %s at %s is not supported", typ, path)
		return reflect.Zero(typ)
	}
}

func assertProjectionMatches(t *testing.T, source, target reflect.Value, excludedTags map[string]struct{}, path, constructor string) {
	t.Helper()

	source = dereferenceValue(source)
	target = dereferenceValue(target)

	require.Truef(t, source.IsValid(), "%s source must be valid", path)
	require.Truef(t, target.IsValid(), "%s target must be valid", path)
	require.Equalf(t, reflect.Struct, source.Kind(), "%s source must be a struct", path)
	require.Equalf(t, reflect.Struct, target.Kind(), "%s target must be a struct", path)

	sourceFields := taggedFields(source.Type())
	targetFields := taggedFields(target.Type())

	var missing []string
	for tag := range sourceFields {
		if _, excluded := excludedTags[tag]; excluded {
			continue
		}
		if _, ok := targetFields[tag]; !ok {
			missing = append(missing, tag)
		}
	}
	sort.Strings(missing)
	require.Emptyf(t, missing, projectionMissingMessage(path, missing, excludedTags))

	var extra []string
	for tag := range targetFields {
		if _, ok := sourceFields[tag]; !ok {
			extra = append(extra, tag)
		}
	}
	sort.Strings(extra)
	require.Emptyf(t, extra, "%s has projected fields without a matching resource tag: %v", path, extra)

	var tags []string
	for tag := range sourceFields {
		if _, excluded := excludedTags[tag]; excluded {
			continue
		}
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	for _, tag := range tags {
		sourceField := source.Field(sourceFields[tag])
		targetField := target.Field(targetFields[tag])
		assertProjectedValueMatches(t, sourceField, targetField, excludedTags, path+"."+tag, constructor)
	}
}

func assertProjectedValueMatches(t *testing.T, source, target reflect.Value, excludedTags map[string]struct{}, path, constructor string) {
	t.Helper()

	if source.Kind() == reflect.Pointer || target.Kind() == reflect.Pointer {
		require.Equalf(t, source.Kind() == reflect.Pointer, target.Kind() == reflect.Pointer, "%s pointer shape changed; update %s to preserve the resource model shape", path, constructor)
		if source.IsNil() || target.IsNil() {
			require.Equalf(t, source.IsNil(), target.IsNil(), "%s nil mismatch; update %s to preserve the resource model value", path, constructor)
			return
		}
		assertProjectedValueMatches(t, source.Elem(), target.Elem(), excludedTags, path, constructor)
		return
	}

	if isTerraformFrameworkValue(source.Type()) || isTerraformFrameworkValue(target.Type()) {
		require.Truef(t, reflect.DeepEqual(source.Interface(), target.Interface()), projectionValueMismatchMessage(path, constructor, source.Interface(), target.Interface()))
		return
	}

	switch source.Kind() {
	case reflect.Struct:
		if len(taggedFields(source.Type())) > 0 || len(taggedFields(target.Type())) > 0 {
			assertProjectionMatches(t, source, target, excludedTags, path, constructor)
			return
		}
		require.Truef(t, reflect.DeepEqual(source.Interface(), target.Interface()), projectionValueMismatchMessage(path, constructor, source.Interface(), target.Interface()))
	case reflect.Slice:
		require.Equalf(t, source.Len(), target.Len(), "%s slice length mismatch; update %s to copy every element for this field", path, constructor)
		for idx := 0; idx < source.Len(); idx++ {
			assertProjectedValueMatches(t, source.Index(idx), target.Index(idx), excludedTags, fmt.Sprintf("%s[%d]", path, idx), constructor)
		}
	case reflect.Map:
		require.Equalf(t, source.Len(), target.Len(), "%s map length mismatch; update %s to copy every key/value for this field", path, constructor)
		for _, key := range source.MapKeys() {
			targetValue := target.MapIndex(key)
			require.Truef(t, targetValue.IsValid(), "%s missing map key %v; update %s to copy this field into the data-source model", path, key.Interface(), constructor)
			assertProjectedValueMatches(t, source.MapIndex(key), targetValue, excludedTags, fmt.Sprintf("%s[%v]", path, key.Interface()), constructor)
		}
	default:
		require.Truef(t, reflect.DeepEqual(source.Interface(), target.Interface()), projectionValueMismatchMessage(path, constructor, source.Interface(), target.Interface()))
	}
}

func projectionMissingMessage(path string, missing []string, excludedTags map[string]struct{}) string {
	return fmt.Sprintf("%s is missing projected fields: %v. Add each field to the data-source model with the same `tfsdk` tag, or add it to the explicit secret-only exclusion list if the omission is intentional.", path, missing)
}

func projectionValueMismatchMessage(path, constructor string, source, target any) string {
	return fmt.Sprintf("%s was not copied by %s. Update %s to project this field from the resource model.\nsource=%#v\ntarget=%#v", path, constructor, constructor, source, target)
}

func taggedFields(typ reflect.Type) map[string]int {
	fields := make(map[string]int)
	for idx := 0; idx < typ.NumField(); idx++ {
		field := typ.Field(idx)
		if !field.IsExported() {
			continue
		}
		tag := field.Tag.Get("tfsdk")
		if tag == "" || tag == "-" {
			continue
		}
		fields[tag] = idx
	}
	return fields
}

func dereferenceValue(value reflect.Value) reflect.Value {
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return value
		}
		value = value.Elem()
	}
	return value
}

func fieldPath(base string, field reflect.StructField) string {
	tag := field.Tag.Get("tfsdk")
	if tag != "" && tag != "-" {
		return base + "." + tag
	}
	return base + "." + field.Name
}

func isTerraformFrameworkValue(typ reflect.Type) bool {
	return typ == tfStringType || typ == tfBoolType || typ == tfInt64Type || typ == tfMapType || typ == tfObjectType
}

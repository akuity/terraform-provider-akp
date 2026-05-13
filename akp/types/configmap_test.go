package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func stringMapValue(m map[string]string) tftypes.Map {
	elems := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elems[k] = tftypes.StringValue(v)
	}
	mv, _ := tftypes.MapValue(tftypes.StringType, elems)
	return mv
}

func TestFilterMapToPlannedKeys(t *testing.T) {
	tests := map[string]struct {
		current  tftypes.Map
		planned  tftypes.Map
		expected tftypes.Map
	}{
		"planned null - returns current unchanged": {
			current:  stringMapValue(map[string]string{"a": "1", "admin.enabled": "true"}),
			planned:  tftypes.MapNull(tftypes.StringType),
			expected: stringMapValue(map[string]string{"a": "1", "admin.enabled": "true"}),
		},
		"planned unknown - returns current unchanged": {
			current:  stringMapValue(map[string]string{"a": "1", "admin.enabled": "true"}),
			planned:  tftypes.MapUnknown(tftypes.StringType),
			expected: stringMapValue(map[string]string{"a": "1", "admin.enabled": "true"}),
		},
		"current null - returns current unchanged": {
			current:  tftypes.MapNull(tftypes.StringType),
			planned:  stringMapValue(map[string]string{"a": "1"}),
			expected: tftypes.MapNull(tftypes.StringType),
		},
		"filters server defaults after create": {
			current:  stringMapValue(map[string]string{"a": "1", "admin.enabled": "true", "server.key": "val"}),
			planned:  stringMapValue(map[string]string{"a": "1"}),
			expected: stringMapValue(map[string]string{"a": "1"}),
		},
		"preserves planned keys with updated values from API": {
			current:  stringMapValue(map[string]string{"a": "normalized", "admin.enabled": "true"}),
			planned:  stringMapValue(map[string]string{"a": "1"}),
			expected: stringMapValue(map[string]string{"a": "normalized"}),
		},
		"preserves state-only keys carried forward by plan modifier": {
			current:  stringMapValue(map[string]string{"a": "1", "admin.enabled": "true", "new_server_key": "val"}),
			planned:  stringMapValue(map[string]string{"a": "1", "admin.enabled": "true"}),
			expected: stringMapValue(map[string]string{"a": "1", "admin.enabled": "true"}),
		},
		"exact match - no filtering needed": {
			current:  stringMapValue(map[string]string{"a": "1", "b": "2"}),
			planned:  stringMapValue(map[string]string{"a": "1", "b": "2"}),
			expected: stringMapValue(map[string]string{"a": "1", "b": "2"}),
		},
		"planned key missing from API - key omitted": {
			current:  stringMapValue(map[string]string{"a": "1"}),
			planned:  stringMapValue(map[string]string{"a": "1", "b": "2"}),
			expected: stringMapValue(map[string]string{"a": "1"}),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			result := FilterMapToPlannedKeys(context.Background(), &diags, tc.current, tc.planned)
			require.False(t, diags.HasError(), "unexpected diagnostics: %s", diags.Errors())
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestToConfigMapTFModel_PreservesResourceCustomizationsYAMLQuoting(t *testing.T) {
	// Issue #11183: the portal API parses `resource.customizations` YAML and re-serializes
	// it on export, which can drop optional YAML quoting around keys like `firstam.net/*`.
	// When the round-tripped value is semantically equivalent to the planned value,
	// ToConfigMapTFModel must preserve the planned value verbatim — otherwise Terraform
	// fails apply with "Provider produced inconsistent result after apply".
	plannedYAML := "'firstam.net/*':\n  health.lua: |\n    return {}\n"
	apiReturnedYAML := "firstam.net/*:\n  health.lua: |\n    return {}\n"

	apiData, err := structpb.NewStruct(map[string]any{
		"resource.customizations": apiReturnedYAML,
	})
	require.NoError(t, err)

	oldCM := stringMapValue(map[string]string{
		"resource.customizations": plannedYAML,
	})

	var diags diag.Diagnostics
	result := ToConfigMapTFModel(context.Background(), &diags, apiData, oldCM)
	require.False(t, diags.HasError(), "unexpected diagnostics: %s", diags.Errors())

	got, ok := result.Elements()["resource.customizations"].(tftypes.String)
	require.True(t, ok, "expected resource.customizations to be a string")
	assert.Equal(t, plannedYAML, got.ValueString(), "planned YAML quoting should be preserved when API roundtrip is semantically equivalent")
}

func TestToConfigMapTFModel_UsesAPIValueWhenResourceCustomizationsDiffer(t *testing.T) {
	// When the API-returned `resource.customizations` contains keys/values that are
	// not present in the planned value, the API value should win so state reflects
	// reality (yamlIsSubset(api, planned) is false).
	plannedYAML := "'firstam.net/Aurora':\n  health.lua: |\n    return {}\n"
	apiReturnedYAML := "firstam.net/Aurora:\n  health.lua: |\n    return {}\nfirstam.net/Extra:\n  health.lua: |\n    return {}\n"

	apiData, err := structpb.NewStruct(map[string]any{
		"resource.customizations": apiReturnedYAML,
	})
	require.NoError(t, err)

	oldCM := stringMapValue(map[string]string{
		"resource.customizations": plannedYAML,
	})

	var diags diag.Diagnostics
	result := ToConfigMapTFModel(context.Background(), &diags, apiData, oldCM)
	require.False(t, diags.HasError(), "unexpected diagnostics: %s", diags.Errors())

	got, ok := result.Elements()["resource.customizations"].(tftypes.String)
	require.True(t, ok, "expected resource.customizations to be a string")
	assert.Equal(t, apiReturnedYAML, got.ValueString(), "API value should win when it has keys not in the planned value")
}

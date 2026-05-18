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
	// it on export, which can drop optional YAML quoting around keys like `somecompany.net/*`.
	// When the round-tripped value is semantically equivalent to the planned value,
	// ToConfigMapTFModel must preserve the planned value verbatim — otherwise Terraform
	// fails apply with "Provider produced inconsistent result after apply".
	plannedYAML := "'somecompany.net/*':\n  health.lua: |\n    return {}\n"
	apiReturnedYAML := "somecompany.net/*:\n  health.lua: |\n    return {}\n"

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

func TestToConfigMapTFModel_StripsIndividualKeysCoveredByCombined(t *testing.T) {
	// When the user's combined `resource.customizations` contains a non-wildcard entry
	// (e.g. `'somecompany.net/Aurora'`), the portal API splits it back into individual
	// `resource.customizations.<resource>.<group>_<kind>` keys on export. Leaving those
	// individual keys in state alongside the preserved combined value would make
	// SuppressNonConfigKeys carry them into the next plan; the subsequent apply would
	// send both forms and the portal would reject the request with
	// "duplicate resources not allowed" because the same group/kind appears twice.
	plannedYAML := "'somecompany.net/Aurora':\n  health.lua: |\n    return {}\n" +
		"'*.crossplane.io/*':\n  health.lua: |\n    return {}\n"
	apiReturnedCombined := "'*.crossplane.io/*':\n  health.lua: |\n    return {}\n"
	apiReturnedIndividual := "hs = {}\nhs.status = \"Healthy\"\nreturn hs\n"

	apiData, err := structpb.NewStruct(map[string]any{
		"resource.customizations":                               apiReturnedCombined,
		"resource.customizations.health.somecompany.net_Aurora": apiReturnedIndividual,
	})
	require.NoError(t, err)

	oldCM := stringMapValue(map[string]string{
		"resource.customizations": plannedYAML,
	})

	var diags diag.Diagnostics
	result := ToConfigMapTFModel(context.Background(), &diags, apiData, oldCM)
	require.False(t, diags.HasError(), "unexpected diagnostics: %s", diags.Errors())

	elems := result.Elements()
	combined, ok := elems["resource.customizations"].(tftypes.String)
	require.True(t, ok, "expected resource.customizations to be a string")
	assert.Equal(t, plannedYAML, combined.ValueString(), "planned combined YAML should be preserved")

	_, hasIndividual := elems["resource.customizations.health.somecompany.net_Aurora"]
	assert.False(t, hasIndividual, "individual key covered by combined YAML must be stripped to avoid duplicate-resource errors on apply")
}

func TestToConfigMapTFModel_StripsIndividualKeysFromStaleState(t *testing.T) {
	// Regression test for the case where TF state already contains both the combined
	// `resource.customizations` and the individual `resource.customizations.<field>.<group_kind>`
	// keys — typically because the state was populated before #11220's strip logic existed,
	// or by an earlier provider version that propagated both forms. On the next refresh
	// after upgrading, ToConfigMapTFModel must self-heal: even though the individual key
	// is present in oldCM, it must be stripped, because keeping it carries the duplicate
	// through `SuppressNonConfigKeys` into the next plan and the subsequent apply is
	// rejected by the portal with "duplicate resources not allowed".
	plannedYAML := "'somecompany.net/Aurora':\n  health.lua: |\n    return {}\n"
	individualVal := "hs = {}\nreturn hs\n"

	apiData, err := structpb.NewStruct(map[string]any{
		"resource.customizations":                               plannedYAML,
		"resource.customizations.health.somecompany.net_Aurora": individualVal,
	})
	require.NoError(t, err)

	oldCM := stringMapValue(map[string]string{
		"resource.customizations":                               plannedYAML,
		"resource.customizations.health.somecompany.net_Aurora": individualVal,
	})

	var diags diag.Diagnostics
	result := ToConfigMapTFModel(context.Background(), &diags, apiData, oldCM)
	require.False(t, diags.HasError(), "unexpected diagnostics: %s", diags.Errors())

	elems := result.Elements()
	assert.Contains(t, elems, "resource.customizations", "combined value must be preserved")
	_, hasIndividual := elems["resource.customizations.health.somecompany.net_Aurora"]
	assert.False(t, hasIndividual, "stale individual key in oldCM must be stripped so the next apply does not resend a duplicate group/kind")
}

func TestToConfigMapTFModel_UsesAPIValueWhenResourceCustomizationsDiffer(t *testing.T) {
	// When the API-returned `resource.customizations` contains keys/values that are
	// not present in the planned value, the API value should win so state reflects
	// reality (yamlIsSubset(api, planned) is false).
	plannedYAML := "'somecompany.net/Aurora':\n  health.lua: |\n    return {}\n"
	apiReturnedYAML := "somecompany.net/Aurora:\n  health.lua: |\n    return {}\nsomecompany.net/Extra:\n  health.lua: |\n    return {}\n"

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

func TestToConfigMapTFModel_KeepsIndividualKeysWhenCombinedDiverges(t *testing.T) {
	// When the API combined YAML diverges from the planned (yamlIsSubset fails), the API
	// value wins and the individual `resource.customizations.<field>.<group_kind>` keys
	// the API also returned are the only place those group/kinds live in state. They must
	// be kept — stripping them here would silently drop state for those resources.
	plannedYAML := "'somecompany.net/Aurora':\n  health.lua: |\n    return {}\n"
	apiCombinedYAML := "somecompany.net/Aurora:\n  health.lua: |\n    return {}\nsomecompany.net/Extra:\n  health.lua: |\n    return {}\n"
	individualVal := "hs = {}\nreturn hs\n"

	apiData, err := structpb.NewStruct(map[string]any{
		"resource.customizations":                               apiCombinedYAML,
		"resource.customizations.health.somecompany.net_Aurora": individualVal,
	})
	require.NoError(t, err)

	oldCM := stringMapValue(map[string]string{
		"resource.customizations": plannedYAML,
	})

	var diags diag.Diagnostics
	result := ToConfigMapTFModel(context.Background(), &diags, apiData, oldCM)
	require.False(t, diags.HasError(), "unexpected diagnostics: %s", diags.Errors())

	elems := result.Elements()
	combined, ok := elems["resource.customizations"].(tftypes.String)
	require.True(t, ok, "expected resource.customizations to be a string")
	assert.Equal(t, apiCombinedYAML, combined.ValueString(), "API combined should win when yamlIsSubset fails")
	assert.Contains(t, elems, "resource.customizations.health.somecompany.net_Aurora",
		"individual key must be preserved when the planned combined is not a superset — it is the only place that group/kind lives in state")
}

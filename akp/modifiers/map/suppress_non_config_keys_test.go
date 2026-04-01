package mapplanmodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stringMapValue(m map[string]string) types.Map {
	elems := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elems[k] = types.StringValue(v)
	}
	mv, _ := types.MapValue(types.StringType, elems)
	return mv
}

func TestSuppressNonConfigKeysModifier(t *testing.T) {
	tests := map[string]struct {
		config   types.Map
		state    types.Map
		plan     types.Map
		expected types.Map
	}{
		"config null with state - preserves state": {
			config:   types.MapNull(types.StringType),
			state:    stringMapValue(map[string]string{"a": "1"}),
			plan:     types.MapUnknown(types.StringType),
			expected: stringMapValue(map[string]string{"a": "1"}),
		},
		"config null state null - no modification": {
			config:   types.MapNull(types.StringType),
			state:    types.MapNull(types.StringType),
			plan:     types.MapUnknown(types.StringType),
			expected: types.MapUnknown(types.StringType),
		},
		"config unknown - no modification": {
			config:   types.MapUnknown(types.StringType),
			state:    stringMapValue(map[string]string{"a": "1"}),
			plan:     types.MapUnknown(types.StringType),
			expected: types.MapUnknown(types.StringType),
		},
		"state null - no modification (fresh create)": {
			config:   stringMapValue(map[string]string{"a": "1"}),
			state:    types.MapNull(types.StringType),
			plan:     stringMapValue(map[string]string{"a": "1"}),
			expected: stringMapValue(map[string]string{"a": "1"}),
		},
		"state unknown - no modification": {
			config:   stringMapValue(map[string]string{"a": "1"}),
			state:    types.MapUnknown(types.StringType),
			plan:     stringMapValue(map[string]string{"a": "1"}),
			expected: stringMapValue(map[string]string{"a": "1"}),
		},
		"state has extra keys - carried forward": {
			config: stringMapValue(map[string]string{"a": "1", "b": "2"}),
			state:  stringMapValue(map[string]string{"a": "1", "b": "2", "server_default1": "x", "server_default2": "y"}),
			plan:   stringMapValue(map[string]string{"a": "1", "b": "2"}),
			expected: stringMapValue(map[string]string{
				"a": "1", "b": "2",
				"server_default1": "x", "server_default2": "y",
			}),
		},
		"config key changed - diff flows through": {
			config:   stringMapValue(map[string]string{"a": "new_value", "b": "2"}),
			state:    stringMapValue(map[string]string{"a": "old_value", "b": "2", "server_default": "x"}),
			plan:     stringMapValue(map[string]string{"a": "new_value", "b": "2"}),
			expected: stringMapValue(map[string]string{"a": "new_value", "b": "2"}),
		},
		"new config key added - diff flows through": {
			config:   stringMapValue(map[string]string{"a": "1", "b": "2", "c": "3"}),
			state:    stringMapValue(map[string]string{"a": "1", "b": "2", "server_default": "x"}),
			plan:     stringMapValue(map[string]string{"a": "1", "b": "2", "c": "3"}),
			expected: stringMapValue(map[string]string{"a": "1", "b": "2", "c": "3"}),
		},
		"config key removed - state preserved (server keeps keys)": {
			config: stringMapValue(map[string]string{"a": "1"}),
			state:  stringMapValue(map[string]string{"a": "1", "b": "2", "server_default": "x"}),
			plan:   stringMapValue(map[string]string{"a": "1"}),
			expected: stringMapValue(map[string]string{
				"a": "1", "b": "2", "server_default": "x",
			}),
		},
		"config subset of state - preserves state": {
			config: stringMapValue(map[string]string{"a": "1"}),
			state:  stringMapValue(map[string]string{"a": "1", "server_default": "x"}),
			plan:   stringMapValue(map[string]string{"a": "1"}),
			expected: stringMapValue(map[string]string{
				"a":              "1",
				"server_default": "x",
			}),
		},
		"exact match - no changes": {
			config:   stringMapValue(map[string]string{"a": "1", "b": "2"}),
			state:    stringMapValue(map[string]string{"a": "1", "b": "2"}),
			plan:     stringMapValue(map[string]string{"a": "1", "b": "2"}),
			expected: stringMapValue(map[string]string{"a": "1", "b": "2"}),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			modifier := SuppressNonConfigKeys()

			req := planmodifier.MapRequest{
				ConfigValue: tc.config,
				StateValue:  tc.state,
				PlanValue:   tc.plan,
			}
			resp := &planmodifier.MapResponse{
				PlanValue: tc.plan,
			}

			modifier.PlanModifyMap(context.Background(), req, resp)

			require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %s", resp.Diagnostics.Errors())
			assert.Equal(t, tc.expected, resp.PlanValue)
		})
	}
}

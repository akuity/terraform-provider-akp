package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

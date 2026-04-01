package bool

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuppressProtobufDefault(t *testing.T) {
	tests := map[string]struct {
		config   types.Bool
		state    types.Bool
		plan     types.Bool
		expected types.Bool
	}{
		"config null, state is false (protobuf default) - suppress diff": {
			config:   types.BoolNull(),
			state:    types.BoolValue(false),
			plan:     types.BoolNull(),
			expected: types.BoolValue(false),
		},
		"config null, state is non-default value - allow diff (removal works)": {
			config:   types.BoolNull(),
			state:    types.BoolValue(true),
			plan:     types.BoolNull(),
			expected: types.BoolNull(),
		},
		"config null, state is null - no modification": {
			config:   types.BoolNull(),
			state:    types.BoolNull(),
			plan:     types.BoolNull(),
			expected: types.BoolNull(),
		},
		"config set - no modification": {
			config:   types.BoolValue(true),
			state:    types.BoolValue(false),
			plan:     types.BoolValue(true),
			expected: types.BoolValue(true),
		},
		"config set to false - no modification (user explicitly set it)": {
			config:   types.BoolValue(false),
			state:    types.BoolValue(false),
			plan:     types.BoolValue(false),
			expected: types.BoolValue(false),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			modifier := SuppressProtobufDefault()
			req := planmodifier.BoolRequest{
				ConfigValue: tc.config,
				StateValue:  tc.state,
				PlanValue:   tc.plan,
			}
			resp := &planmodifier.BoolResponse{
				PlanValue: tc.plan,
			}

			modifier.PlanModifyBool(context.Background(), req, resp)

			require.False(t, resp.Diagnostics.HasError())
			assert.Equal(t, tc.expected, resp.PlanValue)
		})
	}
}

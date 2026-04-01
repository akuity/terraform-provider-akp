package string

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
		config   types.String
		state    types.String
		plan     types.String
		expected types.String
	}{
		"config null, state is empty string (protobuf default) - suppress diff": {
			config:   types.StringNull(),
			state:    types.StringValue(""),
			plan:     types.StringNull(),
			expected: types.StringValue(""),
		},
		"config null, state is non-default value - allow diff (removal works)": {
			config:   types.StringNull(),
			state:    types.StringValue("1.2.3"),
			plan:     types.StringNull(),
			expected: types.StringNull(),
		},
		"config null, state is null - no modification": {
			config:   types.StringNull(),
			state:    types.StringNull(),
			plan:     types.StringNull(),
			expected: types.StringNull(),
		},
		"config set - no modification": {
			config:   types.StringValue("1.2.3"),
			state:    types.StringValue(""),
			plan:     types.StringValue("1.2.3"),
			expected: types.StringValue("1.2.3"),
		},
		"config set to empty string - no modification (user explicitly set it)": {
			config:   types.StringValue(""),
			state:    types.StringValue(""),
			plan:     types.StringValue(""),
			expected: types.StringValue(""),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			modifier := SuppressProtobufDefault()
			req := planmodifier.StringRequest{
				ConfigValue: tc.config,
				StateValue:  tc.state,
				PlanValue:   tc.plan,
			}
			resp := &planmodifier.StringResponse{
				PlanValue: tc.plan,
			}

			modifier.PlanModifyString(context.Background(), req, resp)

			require.False(t, resp.Diagnostics.HasError())
			assert.Equal(t, tc.expected, resp.PlanValue)
		})
	}
}

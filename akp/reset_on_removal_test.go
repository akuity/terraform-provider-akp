//go:build !acc

package akp

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	tfgotypes "github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

func TestRemovalPending(t *testing.T) {
	testCases := map[string]struct {
		config   tftypes.String
		state    tftypes.String
		expected bool
	}{
		"removed from config while state holds a value": {
			config:   tftypes.StringNull(),
			state:    tftypes.StringValue("bundle"),
			expected: true,
		},
		"removed from config, state is the protobuf default": {
			config:   tftypes.StringNull(),
			state:    tftypes.StringValue(""),
			expected: false,
		},
		"removed from config, no prior value": {
			config:   tftypes.StringNull(),
			state:    tftypes.StringNull(),
			expected: false,
		},
		"still configured": {
			config:   tftypes.StringValue("bundle"),
			state:    tftypes.StringValue("other"),
			expected: false,
		},
		"explicitly emptied, which needs no reset": {
			config:   tftypes.StringValue(""),
			state:    tftypes.StringValue("bundle"),
			expected: false,
		},
		"configured with an unresolved expression": {
			config:   tftypes.StringUnknown(),
			state:    tftypes.StringValue("bundle"),
			expected: false,
		},
		"prior value not yet known": {
			config:   tftypes.StringNull(),
			state:    tftypes.StringUnknown(),
			expected: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, removalPending(tc.config, tc.state))
		})
	}
}

// TestConfigUnknown covers what the configuration read cannot report itself:
// an attribute under an unknown object reads back as null.
func TestConfigUnknown(t *testing.T) {
	dataType := tfgotypes.Object{AttributeTypes: map[string]tfgotypes.Type{
		"custom_ca_bundle": tfgotypes.String,
	}}
	specType := tfgotypes.Object{AttributeTypes: map[string]tfgotypes.Type{"data": dataType}}
	rootType := tfgotypes.Object{AttributeTypes: map[string]tfgotypes.Type{"spec": specType}}
	attribute := path.Root("spec").AtName("data").AtName("custom_ca_bundle")

	root := func(spec tfgotypes.Value) tfgotypes.Value {
		return tfgotypes.NewValue(rootType, map[string]tfgotypes.Value{"spec": spec})
	}
	spec := func(data tfgotypes.Value) tfgotypes.Value {
		return tfgotypes.NewValue(specType, map[string]tfgotypes.Value{"data": data})
	}
	data := func(bundle tfgotypes.Value) tfgotypes.Value {
		return tfgotypes.NewValue(dataType, map[string]tfgotypes.Value{"custom_ca_bundle": bundle})
	}

	testCases := map[string]struct {
		raw      tfgotypes.Value
		expected bool
	}{
		"attribute set": {
			raw: root(spec(data(tfgotypes.NewValue(tfgotypes.String, "bundle")))),
		},
		"attribute removed": {
			raw: root(spec(data(tfgotypes.NewValue(tfgotypes.String, nil)))),
		},
		"enclosing block removed": {
			raw: root(spec(tfgotypes.NewValue(dataType, nil))),
		},
		"attribute not yet resolvable": {
			raw:      root(spec(data(tfgotypes.NewValue(tfgotypes.String, tfgotypes.UnknownValue)))),
			expected: true,
		},
		"enclosing block not yet resolvable": {
			raw:      root(spec(tfgotypes.NewValue(dataType, tfgotypes.UnknownValue))),
			expected: true,
		},
		"outermost block not yet resolvable": {
			raw:      root(tfgotypes.NewValue(specType, tfgotypes.UnknownValue)),
			expected: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, configUnknown(tfsdk.Config{Raw: tc.raw}, attribute))
		})
	}
}

func TestResetTo(t *testing.T) {
	testCases := map[string]struct {
		state         tftypes.String
		inherited     string
		expectedValue string
		expectedOK    bool
	}{
		"configured value with nothing to inherit is cleared": {
			state:         tftypes.StringValue("bundle"),
			inherited:     "",
			expectedValue: "",
			expectedOK:    true,
		},
		"value that matches the inherited default is left alone": {
			state:      tftypes.StringValue("instance-default"),
			inherited:  "instance-default",
			expectedOK: false,
		},
		"configured value falls back to the inherited default": {
			state:         tftypes.StringValue("bundle"),
			inherited:     "instance-default",
			expectedValue: "instance-default",
			expectedOK:    true,
		},
		"nothing set on either side": {
			state:      tftypes.StringValue(""),
			inherited:  "",
			expectedOK: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			value, ok := resetTo(tc.state, tc.inherited)
			assert.Equal(t, tc.expectedOK, ok)
			if tc.expectedOK {
				assert.Equal(t, tc.expectedValue, value)
			}
		})
	}
}

// TestResourcesDeclareResetOnRemoval keeps the wiring honest: each resource
// that exposes custom_ca_bundle must reset it on removal, only the child
// resources resolve an inherited default, and the instances also unpin their
// agent version on removal.
func TestResourcesDeclareResetOnRemoval(t *testing.T) {
	caBundle := path.Root("spec").AtName("data").AtName("custom_ca_bundle")

	t.Run("akp_kargo_agent", func(t *testing.T) {
		r, ok := NewAkpKargoAgentResource().(*GenericResource[types.KargoAgent])
		require.True(t, ok)
		require.Len(t, r.ResetOnRemoval, 1)
		assert.Equal(t, caBundle, r.ResetOnRemoval[0].attribute)
		assert.NotNil(t, r.ResetOnRemoval[0].inherited)
	})

	t.Run("akp_cluster", func(t *testing.T) {
		r, ok := NewAkpClusterResource().(*GenericResource[types.Cluster])
		require.True(t, ok)
		require.Len(t, r.ResetOnRemoval, 1)
		assert.Equal(t, caBundle, r.ResetOnRemoval[0].attribute)
		assert.NotNil(t, r.ResetOnRemoval[0].inherited)
	})

	t.Run("akp_kargo_instance", func(t *testing.T) {
		r, ok := NewAkpKargoInstanceResource().(*GenericResource[types.KargoInstance])
		require.True(t, ok)
		spec := path.Root("kargo").AtName("spec").AtName("kargo_instance_spec")
		require.Len(t, r.ResetOnRemoval, 2)
		assert.Equal(t, spec.AtName("agent_customization_defaults").AtName("custom_ca_bundle"), r.ResetOnRemoval[0].attribute)
		assert.Equal(t, spec.AtName("pinned_agent_version"), r.ResetOnRemoval[1].attribute)
		for _, entry := range r.ResetOnRemoval {
			assert.Nil(t, entry.inherited)
		}
	})

	t.Run("akp_instance", func(t *testing.T) {
		r, ok := NewAkpInstanceResource().(*GenericResource[types.Instance])
		require.True(t, ok)
		spec := path.Root("argocd").AtName("spec").AtName("instance_spec")
		require.Len(t, r.ResetOnRemoval, 2)
		assert.Equal(t, spec.AtName("cluster_customization_defaults").AtName("custom_ca_bundle"), r.ResetOnRemoval[0].attribute)
		assert.Equal(t, spec.AtName("pinned_agent_version"), r.ResetOnRemoval[1].attribute)
		for _, entry := range r.ResetOnRemoval {
			assert.Nil(t, entry.inherited)
		}
	})
}

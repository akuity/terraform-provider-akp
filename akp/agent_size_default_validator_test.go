package akp

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

// The validator only reads `<defaults>`, `<defaults>.size` and
// `<defaults>.autoscaler_config`, so these tests drive it through a minimal
// stand-in schema rather than the full instance/Kargo trees. The real paths are
// wired in resource_akp_instance.go and resource_akp_kargo.go.
var (
	testAutoscalerObjectType = tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{"cpu": tftypes.String},
	}
	testDefaultsObjectType = tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"size":              tftypes.String,
			"autoscaler_config": testAutoscalerObjectType,
		},
	}
	testDefaultsRootType = tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{"defaults": testDefaultsObjectType},
	}
)

func testAgentSizeDefaultSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"defaults": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"size": schema.StringAttribute{
						Optional: true,
					},
					"autoscaler_config": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"cpu": schema.StringAttribute{Optional: true},
						},
					},
				},
			},
		},
	}
}

// testAgentSizeDefaults builds the customization-defaults object. A nil size
// means the attribute is null; autoscaler is omitted (null) unless set.
func testAgentSizeDefaults(size tftypes.Value, autoscalerSet bool) tftypes.Value {
	autoscaler := tftypes.NewValue(testAutoscalerObjectType, nil)
	if autoscalerSet {
		autoscaler = tftypes.NewValue(testAutoscalerObjectType, map[string]tftypes.Value{
			"cpu": tftypes.NewValue(tftypes.String, "500m"),
		})
	}
	return tftypes.NewValue(testDefaultsObjectType, map[string]tftypes.Value{
		"size":              size,
		"autoscaler_config": autoscaler,
	})
}

func testAgentSizeDefaultConfig(defaults tftypes.Value) tfsdk.Config {
	return tfsdk.Config{
		Schema: testAgentSizeDefaultSchema(),
		Raw: tftypes.NewValue(testDefaultsRootType, map[string]tftypes.Value{
			"defaults": defaults,
		}),
	}
}

func runAgentSizeDefaultValidator(t *testing.T, defaults tftypes.Value) resource.ValidateConfigResponse {
	t.Helper()
	v := agentSizeDefaultValidator{
		defaultsPath: path.Root("defaults"),
	}
	var resp resource.ValidateConfigResponse
	v.ValidateResource(
		context.Background(),
		resource.ValidateConfigRequest{Config: testAgentSizeDefaultConfig(defaults)},
		&resp,
	)
	return resp
}

func TestAgentSizeDefaultValidatorRejectsScalingLimitsForNonAutoSize(t *testing.T) {
	testCases := map[string]struct {
		size           tftypes.Value
		expectedDetail string
	}{
		"large default carrying scaling limits": {
			size:           tftypes.NewValue(tftypes.String, "large"),
			expectedDetail: "but size is large",
		},
		"small default carrying scaling limits": {
			size:           tftypes.NewValue(tftypes.String, "small"),
			expectedDetail: "but size is small",
		},
		"no default size at all": {
			size:           tftypes.NewValue(tftypes.String, nil),
			expectedDetail: "but size is unset",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			resp := runAgentSizeDefaultValidator(t, testAgentSizeDefaults(tc.size, true))

			require.True(t, resp.Diagnostics.HasError())
			require.Len(t, resp.Diagnostics, 1)
			require.Contains(t, resp.Diagnostics[0].Summary(), `autoscaler_config requires size = "auto"`)
			require.Contains(t, resp.Diagnostics[0].Detail(), tc.expectedDetail)
		})
	}
}

func TestAgentSizeDefaultValidatorAllowsValidCombinations(t *testing.T) {
	testCases := map[string]tftypes.Value{
		// The only size the control plane keeps scaling limits for.
		"auto default with scaling limits": testAgentSizeDefaults(
			tftypes.NewValue(tftypes.String, "auto"), true),
		"large default without scaling limits": testAgentSizeDefaults(
			tftypes.NewValue(tftypes.String, "large"), false),
		"auto default without scaling limits": testAgentSizeDefaults(
			tftypes.NewValue(tftypes.String, "auto"), false),
		"no defaults object at all": tftypes.NewValue(testDefaultsObjectType, nil),
		// A size resolved at apply time cannot be checked yet; the control plane
		// still rejects a bad combination, so deferring here is safe.
		"unknown size with scaling limits": testAgentSizeDefaults(
			tftypes.NewValue(tftypes.String, tftypes.UnknownValue), true),
	}

	for name, defaults := range testCases {
		t.Run(name, func(t *testing.T) {
			resp := runAgentSizeDefaultValidator(t, defaults)
			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics.Errors())
		})
	}
}

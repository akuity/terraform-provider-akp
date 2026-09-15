package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

// argoCDAutoscalerAPIMap is what the server returns for the instance-level
// agent-size default: the quantities have been through resource.Quantity and
// come back from formatMemQuantity's %0.2fGi, so a configured "4Gi" reads as
// "4.00Gi". The whole argocd block is one sensitive attribute, so a mismatch
// surfaces as "inconsistent values for sensitive attribute" with no field name.
func argoCDAutoscalerAPIMap(minMem, maxMem string) map[string]any {
	return map[string]any{
		"spec": map[string]any{
			"instanceSpec": map[string]any{
				"clusterCustomizationDefaults": map[string]any{
					"size": "auto",
					"autoscalerConfig": map[string]any{
						"applicationController": map[string]any{
							"resourceMinimum": map[string]any{"cpu": "500m", "mem": minMem},
							"resourceMaximum": map[string]any{"cpu": "2", "mem": maxMem},
						},
						"repoServer": map[string]any{
							"resourceMinimum": map[string]any{"cpu": "250m", "mem": "0.50Gi"},
							"resourceMaximum": map[string]any{"cpu": "1", "mem": "2.00Gi"},
							"replicaMinimum":  float64(1),
							"replicaMaximum":  float64(3),
						},
					},
				},
			},
		},
	}
}

// planWithArgoCDAutoscaler builds the prior state/plan the operator configured:
// the unnormalised spellings straight from HCL, under the schema's attribute
// names (memory, replicas_*) rather than the API's (mem, replica*).
func planWithArgoCDAutoscaler(minMem, maxMem string) *ArgoCD {
	resourcesType := map[string]attr.Type{"memory": types.StringType, "cpu": types.StringType}
	resources := func(mem, cpu string) types.Object {
		return types.ObjectValueMust(resourcesType, map[string]attr.Value{
			"memory": types.StringValue(mem), "cpu": types.StringValue(cpu),
		})
	}
	appCtrlType := map[string]attr.Type{
		"resource_minimum": types.ObjectType{AttrTypes: resourcesType},
		"resource_maximum": types.ObjectType{AttrTypes: resourcesType},
	}
	repoType := map[string]attr.Type{
		"resource_minimum": types.ObjectType{AttrTypes: resourcesType},
		"resource_maximum": types.ObjectType{AttrTypes: resourcesType},
		"replicas_minimum": types.Int64Type,
		"replicas_maximum": types.Int64Type,
	}
	autoscalerType := map[string]attr.Type{
		"application_controller": types.ObjectType{AttrTypes: appCtrlType},
		"repo_server":            types.ObjectType{AttrTypes: repoType},
	}
	defaultsType := map[string]attr.Type{
		"size":              types.StringType,
		"autoscaler_config": types.ObjectType{AttrTypes: autoscalerType},
	}
	return &ArgoCD{Spec: ArgoCDSpec{InstanceSpec: InstanceSpec{
		ClusterCustomizationDefaults: types.ObjectValueMust(defaultsType, map[string]attr.Value{
			"size": types.StringValue("auto"),
			"autoscaler_config": types.ObjectValueMust(autoscalerType, map[string]attr.Value{
				"application_controller": types.ObjectValueMust(appCtrlType, map[string]attr.Value{
					"resource_minimum": resources(minMem, "500m"),
					"resource_maximum": resources(maxMem, "2"),
				}),
				"repo_server": types.ObjectValueMust(repoType, map[string]attr.Value{
					"resource_minimum": resources("0.5Gi", "250m"),
					"resource_maximum": resources("2Gi", "1"),
					"replicas_minimum": types.Int64Value(1),
					"replicas_maximum": types.Int64Value(3),
				}),
			}),
		}),
	}}}
}

func acdAutoscalerMem(t *testing.T, a *ArgoCD) (string, string) {
	t.Helper()
	ccd := a.Spec.InstanceSpec.ClusterCustomizationDefaults
	require.False(t, ccd.IsNull(), "cluster_customization_defaults not materialised")
	cfg, ok := ccd.Attributes()["autoscaler_config"]
	require.True(t, ok, "autoscaler_config missing")
	s := cfg.String()
	return s, s
}

// The planned spelling must survive the server's normalisation.
func TestArgoCDInstanceAutoscalerQuantitiesKeepPlannedSpelling(t *testing.T) {
	// Mirror Instance.Update: prior state is refreshed in place against a deep
	// copy of itself taken as the plan.
	state := planWithArgoCDAutoscaler("1Gi", "4Gi")
	plan := DeepCopy(state)

	var diags diag.Diagnostics
	diags.Append(BuildStateFromAPI(context.Background(),
		argoCDAutoscalerAPIMap("1.00Gi", "4.00Gi"), state, plan,
		ReverseOverridesMap, ReverseRenamesMap, "argocd")...)
	require.False(t, diags.HasError(), "%v", diags)

	got, _ := acdAutoscalerMem(t, state)
	require.Contains(t, got, `"1Gi"`, "planned min spelling must be preserved, got %s", got)
	require.Contains(t, got, `"4Gi"`, "planned max spelling must be preserved, got %s", got)
	require.NotContains(t, got, "4.00Gi", "normalised form must not reach state")
}

// A genuinely different quantity is drift and must stay visible.
func TestArgoCDInstanceAutoscalerRealDriftStaysVisible(t *testing.T) {
	state := planWithArgoCDAutoscaler("1Gi", "4Gi")
	plan := DeepCopy(state)

	var diags diag.Diagnostics
	diags.Append(BuildStateFromAPI(context.Background(),
		argoCDAutoscalerAPIMap("1.00Gi", "8.00Gi"), state, plan,
		ReverseOverridesMap, ReverseRenamesMap, "argocd")...)
	require.False(t, diags.HasError(), "%v", diags)

	got, _ := acdAutoscalerMem(t, state)
	require.Contains(t, got, "8.00Gi", "real drift must remain visible, got %s", got)
	require.NotContains(t, got, `"4Gi"`, "drift must not be masked by the planned value")
}

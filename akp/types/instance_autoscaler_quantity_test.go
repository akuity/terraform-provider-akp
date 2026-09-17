package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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

// planWithArgoCDAutoscaler builds the prior state/plan the operator configured,
// i.e. the unnormalised spellings straight from HCL.
func planWithArgoCDAutoscaler(t *testing.T, minMem, maxMem string) *ArgoCD {
	t.Helper()
	plan := &ArgoCD{}
	var diags diag.Diagnostics
	diags.Append(BuildStateFromAPI(context.Background(),
		argoCDAutoscalerAPIMap(minMem, maxMem), plan, nil,
		ReverseOverridesMap, ReverseRenamesMap, "argocd")...)
	require.False(t, diags.HasError(), "%v", diags)
	return plan
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
	state := planWithArgoCDAutoscaler(t, "1Gi", "4Gi")
	plan := DeepCopyArgoCD(state)

	var diags diag.Diagnostics
	diags.Append(BuildStateFromAPI(context.Background(),
		argoCDAutoscalerAPIMap("1.00Gi", "4.00Gi"), state, plan,
		ReverseOverridesMap, ReverseRenamesMap, "argocd")...)
	require.False(t, diags.HasError(), "%v", diags)
	preserveInstanceAutoscalerPlanQuantities(state, plan)

	got, _ := acdAutoscalerMem(t, state)
	require.Contains(t, got, `"1Gi"`, "planned min spelling must be preserved, got %s", got)
	require.Contains(t, got, `"4Gi"`, "planned max spelling must be preserved, got %s", got)
	require.NotContains(t, got, "4.00Gi", "normalised form must not reach state")
}

// A genuinely different quantity is drift and must stay visible.
func TestArgoCDInstanceAutoscalerRealDriftStaysVisible(t *testing.T) {
	state := planWithArgoCDAutoscaler(t, "1Gi", "4Gi")
	plan := DeepCopyArgoCD(state)

	var diags diag.Diagnostics
	diags.Append(BuildStateFromAPI(context.Background(),
		argoCDAutoscalerAPIMap("1.00Gi", "8.00Gi"), state, plan,
		ReverseOverridesMap, ReverseRenamesMap, "argocd")...)
	require.False(t, diags.HasError(), "%v", diags)
	preserveInstanceAutoscalerPlanQuantities(state, plan)

	got, _ := acdAutoscalerMem(t, state)
	require.Contains(t, got, "8.00Gi", "real drift must remain visible, got %s", got)
	require.NotContains(t, got, `"4Gi"`, "drift must not be masked by the planned value")
}

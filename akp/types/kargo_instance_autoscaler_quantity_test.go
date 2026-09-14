package types

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

// The server stores the instance-level autoscaler limits as resource.Quantity
// and renders them back through a %0.2fGi formatter, so a configured "4Gi"
// comes back as "4.00Gi" (confirmed against a live export). The whole kargo
// block is one sensitive attribute, so Terraform reports that as
// "inconsistent values for sensitive attribute" and the apply fails without
// naming the field.
func TestPreserveKargoInstanceAutoscalerPlanValues(t *testing.T) {
	mk := func(minMem, minCPU, maxMem, maxCPU string) *Kargo {
		return &Kargo{
			Spec: KargoSpec{
				KargoInstanceSpec: KargoInstanceSpec{
					AgentCustomizationDefaults: &KargoAgentCustomization{
						AutoscalerConfig: &KargoAutoscalerConfig{
							KargoController: &KargoControllerAutoScalingConfig{
								ResourceMinimum: &KargoResources{Mem: types.StringValue(minMem), Cpu: types.StringValue(minCPU)},
								ResourceMaximum: &KargoResources{Mem: types.StringValue(maxMem), Cpu: types.StringValue(maxCPU)},
							},
						},
					},
				},
			},
		}
	}

	state := mk("1.00Gi", "500m", "4.00Gi", "2")
	plan := mk("1Gi", "500m", "4Gi", "2")
	preserveKargoInstanceAutoscalerPlanValues(state, plan)

	ctrl := state.Spec.KargoInstanceSpec.AgentCustomizationDefaults.AutoscalerConfig.KargoController
	require.Equal(t, "1Gi", ctrl.ResourceMinimum.Mem.ValueString(), "planned spelling must survive normalization")
	require.Equal(t, "4Gi", ctrl.ResourceMaximum.Mem.ValueString(), "planned spelling must survive normalization")

	// A genuinely different quantity is drift and must NOT be masked.
	state = mk("1.00Gi", "500m", "8.00Gi", "2")
	plan = mk("1Gi", "500m", "4Gi", "2")
	preserveKargoInstanceAutoscalerPlanValues(state, plan)
	ctrl = state.Spec.KargoInstanceSpec.AgentCustomizationDefaults.AutoscalerConfig.KargoController
	require.Equal(t, "8.00Gi", ctrl.ResourceMaximum.Mem.ValueString(), "real drift must stay visible")
}

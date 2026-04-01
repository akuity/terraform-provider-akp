package akp

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	tfakptypes "github.com/akuity/terraform-provider-akp/akp/types"
)

func TestValidateKargoAgentConfigRejectsNormalizedFieldCombinations(t *testing.T) {
	t.Run("argocd namespace with remote argocd", func(t *testing.T) {
		plan := &tfakptypes.KargoAgent{
			Spec: &tfakptypes.KargoAgentSpec{
				Data: tfakptypes.KargoAgentData{
					RemoteArgocd:    tftypes.StringValue("argocd-id"),
					ArgocdNamespace: tftypes.StringValue("custom-argocd"),
				},
			},
		}

		var diags diag.Diagnostics
		validateKargoAgentConfig(&diags, plan)

		require.True(t, diags.HasError())
		require.Contains(t, diags[0].Summary(), "Invalid argocd_namespace")
	})

	t.Run("maintenance expiry without maintenance mode", func(t *testing.T) {
		plan := &tfakptypes.KargoAgent{
			Spec: &tfakptypes.KargoAgentSpec{
				Data: tfakptypes.KargoAgentData{
					MaintenanceMode:       tftypes.BoolValue(false),
					MaintenanceModeExpiry: tftypes.StringValue("2030-12-31T23:59:59Z"),
				},
			},
		}

		var diags diag.Diagnostics
		validateKargoAgentConfig(&diags, plan)

		require.True(t, diags.HasError())
		require.Contains(t, diags[0].Summary(), "Invalid maintenance_mode_expiry")
	})

	t.Run("maintenance expiry with unknown maintenance mode does not error", func(t *testing.T) {
		plan := &tfakptypes.KargoAgent{
			Spec: &tfakptypes.KargoAgentSpec{
				Data: tfakptypes.KargoAgentData{
					MaintenanceMode:       tftypes.BoolUnknown(),
					MaintenanceModeExpiry: tftypes.StringValue("2030-12-31T23:59:59Z"),
				},
			},
		}

		var diags diag.Diagnostics
		validateKargoAgentConfig(&diags, plan)

		require.False(t, diags.HasError())
	})
}

func TestValidateKargoAgentConfigAllowsValidCombinations(t *testing.T) {
	plan := &tfakptypes.KargoAgent{
		Spec: &tfakptypes.KargoAgentSpec{
			Data: tfakptypes.KargoAgentData{
				RemoteArgocd:          tftypes.StringValue("argocd-id"),
				MaintenanceMode:       tftypes.BoolValue(true),
				MaintenanceModeExpiry: tftypes.StringValue("2030-12-31T23:59:59Z"),
			},
		},
	}

	var diags diag.Diagnostics
	validateKargoAgentConfig(&diags, plan)

	require.False(t, diags.HasError())
}

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

// The control plane rejects `size` for Akuity-managed agents (see
// internal/portalapi/kargo/create_kargo_instance_agent_v1.go). Catch this at
// plan time so users see a clear error instead of a late API failure.
func TestValidateKargoAgentConfigRejectsSizeWithAkuityManaged(t *testing.T) {
	testCases := map[string]struct {
		size          tftypes.String
		akuityManaged tftypes.Bool
		expectError   bool
	}{
		"size set with akuity_managed=true is rejected": {
			size:          tftypes.StringValue("small"),
			akuityManaged: tftypes.BoolValue(true),
			expectError:   true,
		},
		"size omitted with akuity_managed=true is allowed": {
			size:          tftypes.StringNull(),
			akuityManaged: tftypes.BoolValue(true),
			expectError:   false,
		},
		"empty size with akuity_managed=true is allowed": {
			size:          tftypes.StringValue(""),
			akuityManaged: tftypes.BoolValue(true),
			expectError:   false,
		},
		"size set with akuity_managed=false is allowed": {
			size:          tftypes.StringValue("small"),
			akuityManaged: tftypes.BoolValue(false),
			expectError:   false,
		},
		"size omitted with akuity_managed=false is allowed": {
			size:          tftypes.StringNull(),
			akuityManaged: tftypes.BoolValue(false),
			expectError:   false,
		},
		"unknown size with akuity_managed=true is allowed": {
			size:          tftypes.StringUnknown(),
			akuityManaged: tftypes.BoolValue(true),
			expectError:   false,
		},
		"size set with unknown akuity_managed is allowed": {
			size:          tftypes.StringValue("small"),
			akuityManaged: tftypes.BoolUnknown(),
			expectError:   false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			plan := &tfakptypes.KargoAgent{
				Spec: &tfakptypes.KargoAgentSpec{
					Data: tfakptypes.KargoAgentData{
						Size:          tc.size,
						AkuityManaged: tc.akuityManaged,
					},
				},
			}

			var diags diag.Diagnostics
			validateKargoAgentConfig(&diags, plan)

			if tc.expectError {
				require.True(t, diags.HasError())
				require.Contains(t, diags[0].Summary(), "Invalid size")
				return
			}
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
		})
	}
}

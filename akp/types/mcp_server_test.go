package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

// The platform exports a disabled MCP server as "mcpServer": {} (protojson drops the
// false bool, the export type carries omitempty). Refresh must read that as false when
// mcp_server is configured, and must not materialize the object when it is not.
func TestBuildStateFromAPI_MCPServerEnabledFollowsAPI(t *testing.T) {
	cases := map[string]struct {
		prior    *MCPServerConfig
		api      map[string]any
		wantNil  bool
		wantBool bool
	}{
		"enabled in state, disabled out of band": {
			prior:    &MCPServerConfig{Enabled: tftypes.BoolValue(true)},
			api:      map[string]any{"mcpServer": map[string]any{}},
			wantBool: false,
		},
		"enabled in state, disabled explicitly": {
			prior:    &MCPServerConfig{Enabled: tftypes.BoolValue(true)},
			api:      map[string]any{"mcpServer": map[string]any{"enabled": false}},
			wantBool: false,
		},
		"enabled in state and on the platform": {
			prior:    &MCPServerConfig{Enabled: tftypes.BoolValue(true)},
			api:      map[string]any{"mcpServer": map[string]any{"enabled": true}},
			wantBool: true,
		},
		"disabled in state, enabled out of band": {
			prior:    &MCPServerConfig{Enabled: tftypes.BoolValue(false)},
			api:      map[string]any{"mcpServer": map[string]any{"enabled": true}},
			wantBool: true,
		},
		"not configured stays null": {
			prior:   nil,
			api:     map[string]any{"mcpServer": map[string]any{}},
			wantNil: true,
		},
		"not configured, platform reports disabled explicitly": {
			prior:   nil,
			api:     map[string]any{"mcpServer": map[string]any{"enabled": false}},
			wantNil: true,
		},
	}

	for name, tc := range cases {
		t.Run("argocd/"+name, func(t *testing.T) {
			plan := &ArgoCD{Spec: ArgoCDSpec{InstanceSpec: InstanceSpec{McpServer: tc.prior}}}
			state := DeepCopy(plan)
			apiMap := map[string]any{"spec": map[string]any{"instanceSpec": tc.api}}

			var diags diag.Diagnostics
			diags.Append(BuildStateFromAPI(context.Background(), apiMap, state, plan, ReverseOverridesMap, ReverseRenamesMap, "argocd")...)
			require.False(t, diags.HasError(), diags)

			got := state.Spec.InstanceSpec.McpServer
			if tc.wantNil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, tc.wantBool, got.Enabled.ValueBool())
		})
		t.Run("kargo/"+name, func(t *testing.T) {
			plan := &Kargo{Spec: KargoSpec{KargoInstanceSpec: KargoInstanceSpec{McpServer: tc.prior}}}
			state := DeepCopy(plan)
			apiMap := map[string]any{"spec": map[string]any{"kargoInstanceSpec": tc.api}}

			var diags diag.Diagnostics
			diags.Append(BuildStateFromAPI(context.Background(), apiMap, state, plan, KargoReverseOverridesMap, KargoReverseRenamesMap, "kargo")...)
			require.False(t, diags.HasError(), diags)

			got := state.Spec.KargoInstanceSpec.McpServer
			if tc.wantNil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, tc.wantBool, got.Enabled.ValueBool())
		})
	}
}

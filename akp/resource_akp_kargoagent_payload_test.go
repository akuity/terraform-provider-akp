package akp

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	tfakptypes "github.com/akuity/terraform-provider-akp/akp/types"
)

func TestBuildKargoAgentsOmitsNormalizedEmptyFields(t *testing.T) {
	agent := &tfakptypes.KargoAgent{
		Name:        tftypes.StringValue("agent-name"),
		Namespace:   tftypes.StringValue("agent-ns"),
		Labels:      tftypes.MapNull(tftypes.StringType),
		Annotations: tftypes.MapNull(tftypes.StringType),
		Spec: &tfakptypes.KargoAgentSpec{
			Description: tftypes.StringValue("test"),
			Data: tfakptypes.KargoAgentData{
				Size:                  tftypes.StringValue("small"),
				RemoteArgocd:          tftypes.StringValue("argocd-id"),
				AkuityManaged:         tftypes.BoolValue(false),
				MaintenanceMode:       tftypes.BoolValue(false),
				ArgocdNamespace:       tftypes.StringValue(""),
				MaintenanceModeExpiry: tftypes.StringValue(""),
			},
		},
	}

	var diags diag.Diagnostics
	agents := buildKargoAgents(context.Background(), &diags, agent)
	require.False(t, diags.HasError())
	require.Len(t, agents, 1)

	data := agents[0].AsMap()["spec"].(map[string]any)["data"].(map[string]any)
	require.NotContains(t, data, "argocdNamespace")
	require.NotContains(t, data, "maintenanceModeExpiry")
	require.Equal(t, "argocd-id", data["remoteArgocd"])
}

func TestBuildKargoAgentsPreservesConfiguredFields(t *testing.T) {
	agent := &tfakptypes.KargoAgent{
		Name:        tftypes.StringValue("agent-name"),
		Namespace:   tftypes.StringValue("agent-ns"),
		Labels:      tftypes.MapNull(tftypes.StringType),
		Annotations: tftypes.MapNull(tftypes.StringType),
		Spec: &tfakptypes.KargoAgentSpec{
			Description: tftypes.StringValue("test"),
			Data: tfakptypes.KargoAgentData{
				Size:                  tftypes.StringValue("small"),
				MaintenanceMode:       tftypes.BoolValue(true),
				ArgocdNamespace:       tftypes.StringValue("custom-argocd"),
				MaintenanceModeExpiry: tftypes.StringValue("2030-12-31T23:59:59Z"),
			},
		},
	}

	var diags diag.Diagnostics
	agents := buildKargoAgents(context.Background(), &diags, agent)
	require.False(t, diags.HasError())
	require.Len(t, agents, 1)

	data := agents[0].AsMap()["spec"].(map[string]any)["data"].(map[string]any)
	require.Equal(t, "custom-argocd", data["argocdNamespace"])
	require.Equal(t, "2030-12-31T23:59:59Z", data["maintenanceModeExpiry"])
}

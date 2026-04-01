package akp

import (
	"testing"

	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	tfakptypes "github.com/akuity/terraform-provider-akp/akp/types"
)

func TestHydrateKargoAgentFieldsFromExport(t *testing.T) {
	kargoAgent := &tfakptypes.KargoAgent{
		Name: tftypes.StringValue("agent-name"),
		Spec: &tfakptypes.KargoAgentSpec{
			Data: tfakptypes.KargoAgentData{
				ArgocdNamespace:       tftypes.StringValue(""),
				MaintenanceModeExpiry: tftypes.StringValue(""),
			},
		},
	}

	exportedAgent, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{
			"name": "agent-name",
		},
		"spec": map[string]any{
			"data": map[string]any{
				"argocdNamespace":       "custom-argocd",
				"maintenanceModeExpiry": "2030-12-31T23:59:59Z",
			},
		},
	})
	require.NoError(t, err)

	require.True(t, hydrateKargoAgentFieldsFromExport(kargoAgent, exportedAgent))
	require.Equal(t, "custom-argocd", kargoAgent.Spec.Data.ArgocdNamespace.ValueString())
	require.Equal(t, "2030-12-31T23:59:59Z", kargoAgent.Spec.Data.MaintenanceModeExpiry.ValueString())
}

func TestHydrateKargoAgentFieldsFromExportDoesNotOverwriteExistingValues(t *testing.T) {
	kargoAgent := &tfakptypes.KargoAgent{
		Name: tftypes.StringValue("agent-name"),
		Spec: &tfakptypes.KargoAgentSpec{
			Data: tfakptypes.KargoAgentData{
				ArgocdNamespace:       tftypes.StringValue("existing-argocd"),
				MaintenanceModeExpiry: tftypes.StringValue("2040-01-01T00:00:00Z"),
			},
		},
	}

	exportedAgent, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{
			"name": "agent-name",
		},
		"spec": map[string]any{
			"data": map[string]any{
				"argocdNamespace":       "custom-argocd",
				"maintenanceModeExpiry": "2030-12-31T23:59:59Z",
			},
		},
	})
	require.NoError(t, err)

	require.True(t, hydrateKargoAgentFieldsFromExport(kargoAgent, exportedAgent))
	require.Equal(t, "existing-argocd", kargoAgent.Spec.Data.ArgocdNamespace.ValueString())
	require.Equal(t, "2040-01-01T00:00:00Z", kargoAgent.Spec.Data.MaintenanceModeExpiry.ValueString())
}

func TestHydrateKargoAgentFieldsFromExportSupportsProtoStyleShape(t *testing.T) {
	kargoAgent := &tfakptypes.KargoAgent{
		Name: tftypes.StringValue("agent-name"),
		Spec: &tfakptypes.KargoAgentSpec{
			Data: tfakptypes.KargoAgentData{
				ArgocdNamespace:       tftypes.StringValue(""),
				MaintenanceModeExpiry: tftypes.StringValue(""),
			},
		},
	}

	exportedAgent, err := structpb.NewStruct(map[string]any{
		"name": "agent-name",
		"data": map[string]any{
			"argocdNamespace":       "custom-argocd",
			"maintenanceModeExpiry": "2030-12-31T23:59:59Z",
		},
	})
	require.NoError(t, err)

	require.True(t, hydrateKargoAgentFieldsFromExport(kargoAgent, exportedAgent))
	require.Equal(t, "custom-argocd", kargoAgent.Spec.Data.ArgocdNamespace.ValueString())
	require.Equal(t, "2030-12-31T23:59:59Z", kargoAgent.Spec.Data.MaintenanceModeExpiry.ValueString())
}

package akp

import (
	"testing"
	"time"

	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	tfakptypes "github.com/akuity/terraform-provider-akp/akp/types"
)

func TestHydrateKargoAgentFieldsFromGet(t *testing.T) {
	kargoAgent := &tfakptypes.KargoAgent{
		Name: tftypes.StringValue("agent-name"),
		Spec: &tfakptypes.KargoAgentSpec{
			Data: tfakptypes.KargoAgentData{
				ArgocdNamespace:       tftypes.StringValue(""),
				MaintenanceModeExpiry: tftypes.StringValue(""),
			},
		},
	}

	expiry := time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC)
	agent := &kargov1.KargoAgent{
		Name: "agent-name",
		Data: &kargov1.KargoAgentData{
			ArgocdNamespace:       "custom-argocd",
			MaintenanceModeExpiry: timestamppb.New(expiry),
		},
	}

	hydrateKargoAgentFieldsFromGet(kargoAgent, agent)
	require.Equal(t, "custom-argocd", kargoAgent.Spec.Data.ArgocdNamespace.ValueString())
	require.Equal(t, "2030-12-31T23:59:59Z", kargoAgent.Spec.Data.MaintenanceModeExpiry.ValueString())
}

func TestHydrateKargoAgentFieldsFromGetDoesNotOverwriteExistingValues(t *testing.T) {
	kargoAgent := &tfakptypes.KargoAgent{
		Name: tftypes.StringValue("agent-name"),
		Spec: &tfakptypes.KargoAgentSpec{
			Data: tfakptypes.KargoAgentData{
				ArgocdNamespace:       tftypes.StringValue("existing-argocd"),
				MaintenanceModeExpiry: tftypes.StringValue("2040-01-01T00:00:00Z"),
			},
		},
	}

	agent := &kargov1.KargoAgent{
		Name: "agent-name",
		Data: &kargov1.KargoAgentData{
			ArgocdNamespace:       "custom-argocd",
			MaintenanceModeExpiry: timestamppb.New(time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC)),
		},
	}

	hydrateKargoAgentFieldsFromGet(kargoAgent, agent)
	require.Equal(t, "existing-argocd", kargoAgent.Spec.Data.ArgocdNamespace.ValueString())
	require.Equal(t, "2040-01-01T00:00:00Z", kargoAgent.Spec.Data.MaintenanceModeExpiry.ValueString())
}

func TestHydrateKargoAgentFieldsFromGetHandlesMissingData(t *testing.T) {
	kargoAgent := &tfakptypes.KargoAgent{
		Name: tftypes.StringValue("agent-name"),
		Spec: &tfakptypes.KargoAgentSpec{
			Data: tfakptypes.KargoAgentData{
				ArgocdNamespace:       tftypes.StringValue(""),
				MaintenanceModeExpiry: tftypes.StringValue(""),
			},
		},
	}

	// No data at all: fields stay empty, no panic.
	hydrateKargoAgentFieldsFromGet(kargoAgent, &kargov1.KargoAgent{Name: "agent-name"})
	require.Equal(t, "", kargoAgent.Spec.Data.ArgocdNamespace.ValueString())
	require.Equal(t, "", kargoAgent.Spec.Data.MaintenanceModeExpiry.ValueString())

	// Nil receiver pieces: no panic.
	hydrateKargoAgentFieldsFromGet(nil, &kargov1.KargoAgent{})
	hydrateKargoAgentFieldsFromGet(&tfakptypes.KargoAgent{}, &kargov1.KargoAgent{})
	hydrateKargoAgentFieldsFromGet(kargoAgent, nil)
}

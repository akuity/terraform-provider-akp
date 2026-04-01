package types

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
)

func TestKargoAgentUpdate_ClearsBackendNormalizedFields(t *testing.T) {
	agent := &KargoAgent{}
	plan := &KargoAgent{
		Spec: &KargoAgentSpec{
			Description: tftypes.StringValue("self-hosted"),
			Data: KargoAgentData{
				Size:                  tftypes.StringValue("medium"),
				RemoteArgocd:          tftypes.StringValue("argocd-id"),
				ArgocdNamespace:       tftypes.StringValue("custom-argocd"),
				MaintenanceMode:       tftypes.BoolValue(false),
				MaintenanceModeExpiry: tftypes.StringValue("2030-12-31T23:59:59Z"),
			},
		},
	}

	var diags diag.Diagnostics
	agent.Update(context.Background(), &diags, &kargov1.KargoAgent{
		Id:          "agent-id",
		Name:        "agent-name",
		Description: "self-hosted",
		Data: &kargov1.KargoAgentData{
			Namespace:       "test",
			Size:            kargov1.KargoAgentSize_KARGO_AGENT_SIZE_MEDIUM,
			AkuityManaged:   false,
			RemoteArgocd:    "argocd-id",
			MaintenanceMode: func() *bool { v := false; return &v }(),
		},
	}, plan)

	require.False(t, diags.HasError())
	require.NotNil(t, agent.Spec)
	require.Equal(t, "", agent.Spec.Data.ArgocdNamespace.ValueString())
	require.Equal(t, "", agent.Spec.Data.MaintenanceModeExpiry.ValueString())
	require.False(t, agent.Spec.Data.MaintenanceMode.ValueBool())
}

func TestKargoAgentUpdate_UsesAPIValuesFromFullGetResponse(t *testing.T) {
	agent := &KargoAgent{}
	expiry := timestamppb.New(time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC))
	maintenanceMode := true

	var diags diag.Diagnostics
	agent.Update(context.Background(), &diags, &kargov1.KargoAgent{
		Id:          "agent-id",
		Name:        "agent-name",
		Description: "self-hosted",
		Data: &kargov1.KargoAgentData{
			Namespace:             "test",
			Size:                  kargov1.KargoAgentSize_KARGO_AGENT_SIZE_MEDIUM,
			AkuityManaged:         false,
			ArgocdNamespace:       "custom-argocd",
			AllowedJobSa:          []string{"job-runner", "analysis-runner"},
			MaintenanceMode:       &maintenanceMode,
			MaintenanceModeExpiry: expiry,
		},
	}, nil)

	require.False(t, diags.HasError())
	require.NotNil(t, agent.Spec)
	require.Equal(t, "custom-argocd", agent.Spec.Data.ArgocdNamespace.ValueString())
	require.Equal(t, "2030-12-31T23:59:59Z", agent.Spec.Data.MaintenanceModeExpiry.ValueString())
	require.True(t, agent.Spec.Data.MaintenanceMode.ValueBool())
	require.Len(t, agent.Spec.Data.AllowedJobSa, 2)
	require.Equal(t, "job-runner", agent.Spec.Data.AllowedJobSa[0].ValueString())
	require.Equal(t, "analysis-runner", agent.Spec.Data.AllowedJobSa[1].ValueString())
}

func TestKargoAgentUpdate_DefaultsOmittedComputedFieldsForImportParity(t *testing.T) {
	agent := &KargoAgent{}

	var diags diag.Diagnostics
	agent.Update(context.Background(), &diags, &kargov1.KargoAgent{
		Id:          "agent-id",
		Name:        "agent-name",
		Description: "core",
		Data: &kargov1.KargoAgentData{
			Namespace:           "test",
			Size:                kargov1.KargoAgentSize_KARGO_AGENT_SIZE_SMALL,
			AutoUpgradeDisabled: func() *bool { v := true; return &v }(),
			RemoteArgocd:        "argocd-id",
		},
	}, nil)

	require.False(t, diags.HasError())
	require.NotNil(t, agent.Spec)
	require.False(t, agent.Spec.Data.AkuityManaged.ValueBool())
	require.False(t, agent.Spec.Data.MaintenanceMode.ValueBool())
	require.Equal(t, "", agent.Spec.Data.ArgocdNamespace.ValueString())
	require.Equal(t, "", agent.Spec.Data.Kustomization.ValueString())
	require.Equal(t, "", agent.Spec.Data.MaintenanceModeExpiry.ValueString())
}

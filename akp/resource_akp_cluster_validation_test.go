package akp

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	tfakptypes "github.com/akuity/terraform-provider-akp/akp/types"
)

func TestValidateClusterConfigRejectsNormalizedFieldCombinations(t *testing.T) {
	t.Run("maintenance expiry without maintenance mode", func(t *testing.T) {
		plan := &tfakptypes.Cluster{
			Spec: &tfakptypes.ClusterSpec{
				Data: tfakptypes.ClusterData{
					MaintenanceMode:       tftypes.BoolValue(false),
					MaintenanceModeExpiry: tftypes.StringValue("2030-12-31T23:59:59Z"),
				},
			},
		}

		var diags diag.Diagnostics
		validateClusterConfig(&diags, plan)

		require.True(t, diags.HasError())
		require.Contains(t, diags[0].Summary(), "Invalid maintenance_mode_expiry")
	})

	t.Run("maintenance expiry with unknown maintenance mode does not error", func(t *testing.T) {
		plan := &tfakptypes.Cluster{
			Spec: &tfakptypes.ClusterSpec{
				Data: tfakptypes.ClusterData{
					MaintenanceMode:       tftypes.BoolUnknown(),
					MaintenanceModeExpiry: tftypes.StringValue("2030-12-31T23:59:59Z"),
				},
			},
		}

		var diags diag.Diagnostics
		validateClusterConfig(&diags, plan)

		require.False(t, diags.HasError())
	})
}

func TestValidateClusterConfigAllowsValidCombinations(t *testing.T) {
	plan := &tfakptypes.Cluster{
		Spec: &tfakptypes.ClusterSpec{
			Data: tfakptypes.ClusterData{
				MaintenanceMode:       tftypes.BoolValue(true),
				MaintenanceModeExpiry: tftypes.StringValue("2030-12-31T23:59:59Z"),
			},
		},
	}

	var diags diag.Diagnostics
	validateClusterConfig(&diags, plan)

	require.False(t, diags.HasError())
}

func TestPruneNormalizedEmptyClusterFields(t *testing.T) {
	t.Run("drops empty maintenanceModeExpiry", func(t *testing.T) {
		rawMap := map[string]any{
			"data": map[string]any{
				"maintenanceMode":       false,
				"maintenanceModeExpiry": "",
				"size":                  "large",
			},
		}

		pruneNormalizedEmptyClusterFields(rawMap)

		dataMap := rawMap["data"].(map[string]any)
		_, present := dataMap["maintenanceModeExpiry"]
		require.False(t, present, "empty maintenanceModeExpiry must not reach the apply payload")
		require.Equal(t, "large", dataMap["size"], "unrelated fields must be preserved")
		require.Equal(t, false, dataMap["maintenanceMode"], "maintenanceMode must be preserved")
	})

	t.Run("keeps a real maintenanceModeExpiry", func(t *testing.T) {
		rawMap := map[string]any{
			"data": map[string]any{
				"maintenanceMode":       true,
				"maintenanceModeExpiry": "2030-12-31T23:59:59Z",
			},
		}

		pruneNormalizedEmptyClusterFields(rawMap)

		dataMap := rawMap["data"].(map[string]any)
		require.Equal(t, "2030-12-31T23:59:59Z", dataMap["maintenanceModeExpiry"])
	})

	t.Run("tolerates missing or empty input", func(t *testing.T) {
		require.NotPanics(t, func() { pruneNormalizedEmptyClusterFields(nil) })
		require.NotPanics(t, func() { pruneNormalizedEmptyClusterFields(map[string]any{}) })
		require.NotPanics(t, func() {
			pruneNormalizedEmptyClusterFields(map[string]any{"data": map[string]any{}})
		})
	})
}

// Regression test for the actual failure: the pruning must be wired into the
// payload builder, not merely available as a helper. Without the call in
// buildClusters, an empty maintenanceModeExpiry read back from the control
// plane is sent on update and the API rejects the whole request with
// `parsing time "" as "2006-01-02T15:04:05Z07:00"`.
func TestBuildClustersOmitsEmptyMaintenanceModeExpiry(t *testing.T) {
	cluster := &tfakptypes.Cluster{
		Name:        tftypes.StringValue("test-cluster"),
		Namespace:   tftypes.StringValue("akuity"),
		Labels:      tftypes.MapNull(tftypes.StringType),
		Annotations: tftypes.MapNull(tftypes.StringType),
		Spec: &tfakptypes.ClusterSpec{
			Data: tfakptypes.ClusterData{
				Size: tftypes.StringValue("large"),
				// What the control plane returns whenever maintenance mode is
				// off, which is the default state for every cluster.
				MaintenanceMode:       tftypes.BoolValue(false),
				MaintenanceModeExpiry: tftypes.StringValue(""),
			},
		},
	}

	var diagnostics diag.Diagnostics
	structs := buildClusters(t.Context(), &diagnostics, cluster)

	require.False(t, diagnostics.HasError(), "unexpected diagnostics: %v", diagnostics)
	require.Len(t, structs, 1)

	payload := structs[0].AsMap()
	spec, ok := payload["spec"].(map[string]any)
	require.True(t, ok, "payload missing spec: %v", payload)
	data, ok := spec["data"].(map[string]any)
	require.True(t, ok, "spec missing data: %v", spec)

	_, present := data["maintenanceModeExpiry"]
	require.False(t, present,
		"empty maintenanceModeExpiry must be omitted from the apply payload; the API rejects \"\" as a timestamp")
}

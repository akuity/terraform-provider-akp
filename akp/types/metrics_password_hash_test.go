package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

// The platform never returns the metrics password hash, so refresh must keep the configured one.
func TestBuildStateFromAPI_MetricsPasswordHashKeepsPlan(t *testing.T) {
	plan := &ArgoCD{Spec: ArgoCDSpec{InstanceSpec: InstanceSpec{
		MetricsIngressUsername:     tftypes.StringValue("metrics"),
		MetricsIngressPasswordHash: tftypes.StringValue("$2a$10$stored"),
	}}}
	state := DeepCopy(plan)
	apiMap := map[string]any{"spec": map[string]any{"instanceSpec": map[string]any{"metricsIngressUsername": "metrics"}}}

	var diags diag.Diagnostics
	diags.Append(BuildStateFromAPI(context.Background(), apiMap, state, plan, ReverseOverridesMap, ReverseRenamesMap, "argocd")...)
	require.False(t, diags.HasError(), diags)
	require.Equal(t, "metrics", state.Spec.InstanceSpec.MetricsIngressUsername.ValueString())
	require.Equal(t, "$2a$10$stored", state.Spec.InstanceSpec.MetricsIngressPasswordHash.ValueString())
}

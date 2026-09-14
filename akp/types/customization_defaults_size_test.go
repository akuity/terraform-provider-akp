package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stretchr/testify/require"
)

// The instance-level default agent size is returned by the API as a proto enum
// name. Both instance types keep it in a block Terraform marks sensitive, so a
// missing reverse override does not surface as a readable diff: the apply fails
// with "inconsistent values for sensitive attribute", because the plan holds
// "auto" and the post-apply state holds "CLUSTER_SIZE_AUTO".
//
// connectivity is asserted alongside size in both cases because it is the
// neighbouring field that already had the override — it is what makes a
// regression here legible as "size specifically", not "the whole block".
func TestCustomizationDefaultsSizeIsNormalizedFromProtoEnum(t *testing.T) {
	t.Run("kargo", func(t *testing.T) {
		apiMap := map[string]any{
			"spec": map[string]any{
				"kargoInstanceSpec": map[string]any{
					"agentCustomizationDefaults": map[string]any{
						"size":         "KARGO_AGENT_SIZE_AUTO",
						"connectivity": "CONNECTIVITY_PUBLIC",
					},
				},
			},
		}
		k := &Kargo{}
		var diags diag.Diagnostics
		diags.Append(BuildStateFromAPI(context.Background(), apiMap, k, nil, KargoReverseOverridesMap, KargoReverseRenamesMap, "kargo")...)
		require.False(t, diags.HasError(), "%v", diags)

		require.NotNil(t, k.Spec.KargoInstanceSpec)
		acd := k.Spec.KargoInstanceSpec.AgentCustomizationDefaults
		require.NotNil(t, acd)
		require.Equal(t, "auto", acd.Size.ValueString())
		require.Equal(t, "public", acd.Connectivity.ValueString())
	})

	t.Run("argocd", func(t *testing.T) {
		apiMap := map[string]any{
			"spec": map[string]any{
				"instanceSpec": map[string]any{
					"clusterCustomizationDefaults": map[string]any{
						"size":         "CLUSTER_SIZE_AUTO",
						"connectivity": "CONNECTIVITY_PUBLIC",
					},
				},
			},
		}
		a := &ArgoCD{}
		var diags diag.Diagnostics
		diags.Append(BuildStateFromAPI(context.Background(), apiMap, a, nil, ReverseOverridesMap, ReverseRenamesMap, "spec")...)
		require.False(t, diags.HasError(), "%v", diags)

		ccd := a.Spec.InstanceSpec.ClusterCustomizationDefaults
		require.False(t, ccd.IsNull(), "cluster_customization_defaults was not materialized")
		attrs := ccd.Attributes()
		require.Contains(t, attrs, "size")
		require.Equal(t, `"auto"`, attrs["size"].String())
		require.Equal(t, `"public"`, attrs["connectivity"].String())
	})
}

package akp

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// The instance-level agent-size default reuses the per-cluster attribute names
// (`memory`, `replicas_*`) while the API expects `mem`/`replica*`. Two things must
// hold, and each has broken once:
//
//   - types.RenamesMap needs entries under the instance path, and
//   - buildArgoCD has to actually PASS that map. It passed nil, so the entries
//     could never fire and UpdateInstance rejected the patch with
//     `unknown field "memory"`.
//
// This goes through buildArgoCD rather than the converter directly, so it covers
// the wiring as well as the entries. The acceptance tests catch this only by
// running a real apply against a live API.
func TestBuildArgoCDRenamesAgentSizeDefaultAutoscaler(t *testing.T) {
	resourcesType := map[string]attr.Type{"memory": tftypes.StringType, "cpu": tftypes.StringType}
	resources := func(mem, cpu string) tftypes.Object {
		return tftypes.ObjectValueMust(resourcesType, map[string]attr.Value{
			"memory": tftypes.StringValue(mem), "cpu": tftypes.StringValue(cpu),
		})
	}
	appCtrlType := map[string]attr.Type{
		"resource_minimum": tftypes.ObjectType{AttrTypes: resourcesType},
		"resource_maximum": tftypes.ObjectType{AttrTypes: resourcesType},
	}
	repoType := map[string]attr.Type{
		"resource_minimum": tftypes.ObjectType{AttrTypes: resourcesType},
		"resource_maximum": tftypes.ObjectType{AttrTypes: resourcesType},
		"replicas_minimum": tftypes.Int64Type,
		"replicas_maximum": tftypes.Int64Type,
	}
	autoscalerType := map[string]attr.Type{
		"application_controller": tftypes.ObjectType{AttrTypes: appCtrlType},
		"repo_server":            tftypes.ObjectType{AttrTypes: repoType},
	}
	defaultsType := map[string]attr.Type{
		"size":              tftypes.StringType,
		"autoscaler_config": tftypes.ObjectType{AttrTypes: autoscalerType},
	}

	instance := &types.Instance{
		ArgoCD: &types.ArgoCD{
			Spec: types.ArgoCDSpec{
				InstanceSpec: types.InstanceSpec{
					ClusterCustomizationDefaults: tftypes.ObjectValueMust(defaultsType, map[string]attr.Value{
						"size": tftypes.StringValue("auto"),
						"autoscaler_config": tftypes.ObjectValueMust(autoscalerType, map[string]attr.Value{
							"application_controller": tftypes.ObjectValueMust(appCtrlType, map[string]attr.Value{
								"resource_minimum": resources("1Gi", "500m"),
								"resource_maximum": resources("16Gi", "6000m"),
							}),
							"repo_server": tftypes.ObjectValueMust(repoType, map[string]attr.Value{
								"resource_minimum": resources("0.5Gi", "250m"),
								"resource_maximum": resources("6Gi", "4000m"),
								"replicas_minimum": tftypes.Int64Value(2),
								"replicas_maximum": tftypes.Int64Value(10),
							}),
						}),
					}),
				},
			},
		},
	}

	var diags diag.Diagnostics
	built := buildArgoCD(context.Background(), &diags, instance)
	require.False(t, diags.HasError(), "buildArgoCD reported: %v", diags)
	require.NotNil(t, built)

	dig := func(m map[string]any, keys ...string) map[string]any {
		t.Helper()
		for _, k := range keys {
			next, ok := m[k].(map[string]any)
			require.Truef(t, ok, "missing %q in %#v", k, m)
			m = next
		}
		return m
	}

	ac := dig(built.AsMap(), "spec", "instanceSpec", "clusterCustomizationDefaults", "autoscalerConfig")

	appMin := dig(ac, "applicationController", "resourceMinimum")
	require.Equal(t, "1Gi", appMin["mem"], "memory must be renamed to mem")
	require.NotContains(t, appMin, "memory", "the API rejects the Terraform name")

	repo := dig(ac, "repoServer")
	require.Equal(t, "0.5Gi", dig(repo, "resourceMinimum")["mem"])
	require.NotContains(t, dig(repo, "resourceMinimum"), "memory")
	require.EqualValues(t, 2, repo["replicaMinimum"], "replicas_minimum must be renamed")
	require.EqualValues(t, 10, repo["replicaMaximum"], "replicas_maximum must be renamed")
}

package types

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
)

func TestToDataSourceConfigMapTFModel_ReturnsAPIValuesWhenStateEmpty(t *testing.T) {
	ctx := context.Background()
	apiMap, err := structpb.NewStruct(map[string]any{
		"exec.enabled":     "true",
		"helm.enabled":     "true",
		"accounts.alice":   "login",
		"resource.compare": "ignore",
		"resource.customizations": `
argoproj.io/Application:
  health.lua: |
    hs = {}
`,
	})
	require.NoError(t, err)

	var diags diag.Diagnostics
	got := ToDataSourceConfigMapTFModel(ctx, &diags, apiMap, tftypes.MapNull(tftypes.StringType))
	require.False(t, diags.HasError())

	var values map[string]string
	diags.Append(got.ElementsAs(ctx, &values, true)...)
	require.False(t, diags.HasError())
	require.Equal(t, "true", values["exec.enabled"])
	require.Equal(t, "true", values["helm.enabled"])
	require.Equal(t, "login", values["accounts.alice"])
}

func TestSyncResources_DataSourceSerializesJSONAndNormalizesArgoKeys(t *testing.T) {
	ctx := context.Background()

	defaultProject, err := structpb.NewStruct(map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "AppProject",
		"metadata": map[string]any{
			"name":      "default",
			"namespace": "argocd",
		},
	})
	require.NoError(t, err)

	managedProject, err := structpb.NewStruct(map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "AppProject",
		"metadata": map[string]any{
			"name":      "test-project",
			"namespace": "argocd",
		},
		"spec": map[string]any{
			"description": "Test project",
		},
	})
	require.NoError(t, err)

	var diags diag.Diagnostics
	got, err := syncResources(
		ctx,
		&diags,
		tftypes.MapNull(tftypes.StringType),
		[]*structpb.Struct{defaultProject, managedProject},
		"ArgoCD",
		true,
	)
	require.NoError(t, err)
	require.False(t, diags.HasError())

	var values map[string]string
	diags.Append(got.ElementsAs(ctx, &values, true)...)
	require.False(t, diags.HasError())
	require.Len(t, values, 1)

	expectedJSON, err := json.Marshal(managedProject.AsMap())
	require.NoError(t, err)
	require.JSONEq(t, string(expectedJSON), values["argoproj.io/v1alpha1/AppProject//test-project"])
}

func TestClusterUpdate_DataSourceHydratesAutoAgentSizeConfig(t *testing.T) {
	cluster := &Cluster{
		InstanceID: tftypes.StringValue("instance-id"),
		Name:       tftypes.StringValue("cluster-auto"),
	}

	apiCluster := &argocdv1.Cluster{
		Id:   "cluster-id",
		Name: "cluster-auto",
		Data: &argocdv1.ClusterData{
			Namespace: "argocd",
			Size:      argocdv1.ClusterSize_CLUSTER_SIZE_AUTO,
			AutoscalerConfig: &argocdv1.AutoScalerConfig{
				ApplicationController: &argocdv1.AppControllerAutoScalingConfig{
					ResourceMinimum: &argocdv1.Resources{Cpu: "250m", Mem: "1Gi"},
					ResourceMaximum: &argocdv1.Resources{Cpu: "3", Mem: "2Gi"},
				},
				RepoServer: &argocdv1.RepoServerAutoScalingConfig{
					ResourceMinimum: &argocdv1.Resources{Cpu: "250m", Mem: "256Mi"},
					ResourceMaximum: &argocdv1.Resources{Cpu: "3", Mem: "2Gi"},
					ReplicaMinimum:  1,
					ReplicaMaximum:  3,
				},
			},
		},
	}

	var diags diag.Diagnostics
	cluster.Update(context.Background(), &diags, apiCluster, nil)
	require.False(t, diags.HasError())
	require.Equal(t, "auto", cluster.Spec.Data.Size.ValueString())
	require.NotNil(t, cluster.Spec.Data.AutoscalerConfig)
	require.Equal(t, "3", cluster.Spec.Data.AutoscalerConfig.ApplicationController.ResourceMaximum.Cpu.ValueString())
	require.Equal(t, int64(3), cluster.Spec.Data.AutoscalerConfig.RepoServer.ReplicasMaximum.ValueInt64())
}

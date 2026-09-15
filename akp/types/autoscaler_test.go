//go:build !acc

package types

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
)

func TestToAutoScalerConfigTFModel(t *testing.T) {
	api := &argocdv1.AutoScalerConfig{
		ApplicationController: &argocdv1.AppControllerAutoScalingConfig{
			ResourceMinimum: &argocdv1.Resources{Mem: "1024Mi", Cpu: "500m"},
		},
		RepoServer: &argocdv1.RepoServerAutoScalingConfig{
			ResourceMaximum: &argocdv1.Resources{Mem: "2Gi", Cpu: "1"},
			ReplicaMinimum:  1,
			ReplicaMaximum:  3,
		},
	}
	planned := &AutoScalerConfig{
		ApplicationController: &AppControllerAutoScalingConfig{
			ResourceMinimum: &Resources{Memory: types.StringValue("1Gi"), Cpu: types.StringValue("0.5")},
		},
		RepoServer: &RepoServerAutoScalingConfig{ReplicasMaximum: types.Int64Value(5)},
	}
	cluster := func(size string, cfg *AutoScalerConfig) *Cluster {
		return &Cluster{Spec: &ClusterSpec{Data: ClusterData{Size: types.StringValue(size), AutoscalerConfig: cfg}}}
	}

	require.Nil(t, toAutoScalerConfigTFModel(nil, nil))
	require.Nil(t, toAutoScalerConfigTFModel(cluster("auto", nil), api))
	require.Same(t, planned, toAutoScalerConfigTFModel(cluster("small", planned), api))

	got := toAutoScalerConfigTFModel(nil, api)
	require.Equal(t, "1024Mi", got.ApplicationController.ResourceMinimum.Memory.ValueString())
	require.Nil(t, got.ApplicationController.ResourceMaximum)
	require.Equal(t, int64(3), got.RepoServer.ReplicasMaximum.ValueInt64())

	got = toAutoScalerConfigTFModel(cluster("auto", planned), api)
	require.Equal(t, "1Gi", got.ApplicationController.ResourceMinimum.Memory.ValueString())
	require.Equal(t, "0.5", got.ApplicationController.ResourceMinimum.Cpu.ValueString())
	require.Equal(t, "2Gi", got.RepoServer.ResourceMaximum.Memory.ValueString())
	require.Equal(t, int64(1), got.RepoServer.ReplicasMinimum.ValueInt64())
	require.Equal(t, int64(5), got.RepoServer.ReplicasMaximum.ValueInt64())
}

package types

import (
	"context"
	"encoding/json"
	"reflect"
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

// TestClusterUpdate_ConnectivityNormalizesEnum guards the connectivity round-trip.
// protojson serializes the API enum to its proto name ("CONNECTIVITY_PUBLIC"); the TF
// schema stores the lowercase "public"/"private" form, and an unset value must resolve
// to the documented "public" default so that create, refresh, and import all agree.
func TestClusterUpdate_ConnectivityNormalizesEnum(t *testing.T) {
	newCluster := func() *Cluster {
		return &Cluster{
			InstanceID: tftypes.StringValue("instance-id"),
			Name:       tftypes.StringValue("cluster-conn"),
		}
	}
	newAPICluster := func(c argocdv1.Connectivity) *argocdv1.Cluster {
		return &argocdv1.Cluster{
			Id:   "cluster-id",
			Name: "cluster-conn",
			Data: &argocdv1.ClusterData{
				Namespace:    "argocd",
				Size:         argocdv1.ClusterSize_CLUSTER_SIZE_SMALL,
				Connectivity: c,
			},
		}
	}

	testCases := map[string]struct {
		apiValue   argocdv1.Connectivity
		readImport bool
		expected   string
	}{
		"public enum maps to public":               {apiValue: argocdv1.Connectivity_CONNECTIVITY_PUBLIC, expected: "public"},
		"private enum maps to private":             {apiValue: argocdv1.Connectivity_CONNECTIVITY_PRIVATE, expected: "private"},
		"unspecified defaults to public on apply":  {apiValue: argocdv1.Connectivity_CONNECTIVITY_UNSPECIFIED, expected: "public"},
		"unspecified defaults to public on import": {apiValue: argocdv1.Connectivity_CONNECTIVITY_UNSPECIFIED, readImport: true, expected: "public"},
		"private enum maps to private on import":   {apiValue: argocdv1.Connectivity_CONNECTIVITY_PRIVATE, readImport: true, expected: "private"},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			if tc.readImport {
				ctx = WithReadContext(ctx)
			}
			cluster := newCluster()
			var diags diag.Diagnostics
			cluster.Update(ctx, &diags, newAPICluster(tc.apiValue), nil)
			require.False(t, diags.HasError())
			require.Equal(t, tc.expected, cluster.Spec.Data.Connectivity.ValueString())
		})
	}
}

// TestConnectivityReverseOverride_HandlesBothRepresentations guards the override against
// both API forms: the protojson enum name (cluster/kargo-agent read path) and the
// lowercase v1alpha1 export string (instance/kargo-instance read path), plus the
// omitted/unset case that must default to "public".
func TestConnectivityReverseOverride_HandlesBothRepresentations(t *testing.T) {
	override := ConnectivityReverseOverride()
	testCases := map[string]struct {
		mapValue any
		expected string
	}{
		"proto public":     {mapValue: "CONNECTIVITY_PUBLIC", expected: "public"},
		"proto private":    {mapValue: "CONNECTIVITY_PRIVATE", expected: "private"},
		"export public":    {mapValue: "public", expected: "public"},
		"export private":   {mapValue: "private", expected: "private"},
		"missing":          {mapValue: nil, expected: "public"},
		"empty string":     {mapValue: "", expected: "public"},
		"unspecified enum": {mapValue: "CONNECTIVITY_UNSPECIFIED", expected: "public"},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			val, handled := override(tc.mapValue, reflect.Value{})
			require.True(t, handled)
			str, ok := val.(tftypes.String)
			require.True(t, ok)
			require.Equal(t, tc.expected, str.ValueString())
		})
	}
}

// TestConnectivityProtoToTF_CustomizationDefaultsOverride guards the override used for the
// optional customization-defaults objects: it must normalize both the enum name and the
// lowercase form, but decline (so the object is not materialized) when the value is absent
// or unset. This is the path that otherwise leaks "CONNECTIVITY_PUBLIC" into a sensitive
// block on export.
func TestConnectivityProtoToTF_CustomizationDefaultsOverride(t *testing.T) {
	override := ProtoEnumToLowerString(connectivityProtoToTF)

	mapped := map[string]string{
		"CONNECTIVITY_PUBLIC":  "public",
		"CONNECTIVITY_PRIVATE": "private",
		"public":               "public",
		"private":              "private",
	}
	for in, want := range mapped {
		t.Run("maps "+in, func(t *testing.T) {
			val, handled := override(in, reflect.Value{})
			require.True(t, handled)
			str, ok := val.(tftypes.String)
			require.True(t, ok)
			require.Equal(t, want, str.ValueString())
		})
	}

	for name, in := range map[string]any{"missing": nil, "empty": "", "unspecified": "CONNECTIVITY_UNSPECIFIED"} {
		t.Run("declines "+name, func(t *testing.T) {
			_, handled := override(in, reflect.Value{})
			require.False(t, handled)
		})
	}
}

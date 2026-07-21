package akp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
)

// TestKargoSupportedGroupKinds_CustomPromotionStep guards the TF side of issue #11154:
// without this entry, isKargoResourceValid would reject the kind as "unsupported" and
// TF apply would fail for users declaring a CustomPromotionStep in their manifests.
func TestKargoSupportedGroupKinds_CustomPromotionStep(t *testing.T) {
	gk := schema.GroupKind{Group: "ee.kargo.akuity.io", Kind: "CustomPromotionStep"}
	_, ok := kargoSupportedGroupKinds[gk]
	require.True(t, ok, "CustomPromotionStep missing from kargoSupportedGroupKinds")

	un := &unstructured.Unstructured{}
	un.SetUnstructuredContent(map[string]any{
		"apiVersion": "ee.kargo.akuity.io/v1alpha1",
		"kind":       "CustomPromotionStep",
		"metadata":   map[string]any{"name": "kyverno-validate"},
	})
	require.NoError(t, isKargoResourceValid(un))
}

// TestKargoResourceGroups_CustomPromotionStep ensures the TF apply path appends
// CustomPromotionStep objects to ApplyKargoInstanceRequest.CustomPromotionSteps.
// Symmetric to the CLI fix; before this, TF sync silently dropped the resource.
func TestKargoResourceGroups_CustomPromotionStep(t *testing.T) {
	g, ok := kargoResourceGroups["CustomPromotionStep"]
	require.True(t, ok, "CustomPromotionStep missing from kargoResourceGroups")

	s, err := structpb.NewStruct(map[string]any{"k": "v"})
	require.NoError(t, err)

	req := &kargov1.ApplyKargoInstanceRequest{}
	g.appendFunc(req, s)
	assert.Equal(t, []*structpb.Struct{s}, req.CustomPromotionSteps)
}

// TestKargoSupportedGroupKinds_ClusterConfig guards the TF side of cluster-wide config
// support: without this entry, isKargoResourceValid would reject ClusterConfig as
// "unsupported" and TF apply would fail for users declaring one in kargo_resources.
func TestKargoSupportedGroupKinds_ClusterConfig(t *testing.T) {
	gk := schema.GroupKind{Group: "kargo.akuity.io", Kind: "ClusterConfig"}
	_, ok := kargoSupportedGroupKinds[gk]
	require.True(t, ok, "ClusterConfig missing from kargoSupportedGroupKinds")

	un := &unstructured.Unstructured{}
	un.SetUnstructuredContent(map[string]any{
		"apiVersion": "kargo.akuity.io/v1alpha1",
		"kind":       "ClusterConfig",
		"metadata":   map[string]any{"name": "cluster"},
	})
	require.NoError(t, isKargoResourceValid(un))
}

// TestKargoResourceGroups_ClusterConfig ensures the TF apply path appends ClusterConfig
// objects to ApplyKargoInstanceRequest.ClusterConfigs.
func TestKargoResourceGroups_ClusterConfig(t *testing.T) {
	g, ok := kargoResourceGroups["ClusterConfig"]
	require.True(t, ok, "ClusterConfig missing from kargoResourceGroups")

	s, err := structpb.NewStruct(map[string]any{"k": "v"})
	require.NoError(t, err)

	req := &kargov1.ApplyKargoInstanceRequest{}
	g.appendFunc(req, s)
	assert.Equal(t, []*structpb.Struct{s}, req.ClusterConfigs)
}

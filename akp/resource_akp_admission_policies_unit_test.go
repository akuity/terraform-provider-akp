package akp

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

func admissionResource(apiVersion, kind string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata":   map[string]any{"name": "policy"},
	}}
}

func TestArgoResourceValid_AdmissionPolicies(t *testing.T) {
	testCases := map[string]struct {
		obj   *unstructured.Unstructured
		valid bool
	}{
		"policy":                 {obj: admissionResource("admissionregistration.k8s.io/v1", "ValidatingAdmissionPolicy"), valid: true},
		"binding":                {obj: admissionResource("admissionregistration.k8s.io/v1", "ValidatingAdmissionPolicyBinding"), valid: true},
		"policy in beta version": {obj: admissionResource("admissionregistration.k8s.io/v1beta1", "ValidatingAdmissionPolicy")},
		"webhook configuration":  {obj: admissionResource("admissionregistration.k8s.io/v1", "ValidatingWebhookConfiguration")},
		"application":            {obj: admissionResource("argoproj.io/v1alpha1", "Application"), valid: true},
		"application in admissionregistration version": {obj: admissionResource("admissionregistration.k8s.io/v1", "Application")},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := isArgoResourceValid(tc.obj)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestArgoResourceGroups_AdmissionPolicies(t *testing.T) {
	s, err := structpb.NewStruct(map[string]any{"k": "v"})
	require.NoError(t, err)

	req := &argocdv1.ApplyInstanceRequest{}
	for _, kind := range []string{"ValidatingAdmissionPolicy", "ValidatingAdmissionPolicyBinding"} {
		g, ok := argoResourceGroups[kind]
		require.True(t, ok, kind)
		assert.Equal(t, "admissionregistration.k8s.io", g.group)
		*g.slice(req) = append(*g.slice(req), s)
	}
	assert.Equal(t, []*structpb.Struct{s}, req.ValidatingAdmissionPolicies)
	assert.Equal(t, []*structpb.Struct{s}, req.ValidatingAdmissionPolicyBindings)
}

func TestKargoResourceGroups_AdmissionPolicies(t *testing.T) {
	s, err := structpb.NewStruct(map[string]any{"k": "v"})
	require.NoError(t, err)

	req := &kargov1.ApplyKargoInstanceRequest{}
	for _, kind := range []string{"ValidatingAdmissionPolicy", "ValidatingAdmissionPolicyBinding"} {
		g, ok := kargoResourceGroups[kind]
		require.True(t, ok, kind)
		assert.Equal(t, "admissionregistration.k8s.io", g.group)
		require.NoError(t, isKargoResourceValid(admissionResource("admissionregistration.k8s.io/v1", kind)))
		require.Error(t, isKargoResourceValid(admissionResource("admissionregistration.k8s.io/v1beta1", kind)))
		*g.slice(req) = append(*g.slice(req), s)
	}
	assert.Equal(t, []*structpb.Struct{s}, req.ValidatingAdmissionPolicies)
	assert.Equal(t, []*structpb.Struct{s}, req.ValidatingAdmissionPolicyBindings)
}

func resourcesMap(t *testing.T, manifests ...string) tftypes.Map {
	t.Helper()
	elements := map[string]attr.Value{}
	for i, m := range manifests {
		elements[fmt.Sprintf("key-%d", i)] = tftypes.StringValue(m)
	}
	value, diags := tftypes.MapValue(tftypes.StringType, elements)
	require.False(t, diags.HasError())
	return value
}

const (
	testPolicyManifest = `{"apiVersion":"admissionregistration.k8s.io/v1","kind":"ValidatingAdmissionPolicy","metadata":{"name":"p"}}`
	testAppManifest    = `{"apiVersion":"argoproj.io/v1alpha1","kind":"Application","metadata":{"name":"app"}}`
)

func TestManagesAdmissionPolicies(t *testing.T) {
	ctx := context.Background()
	assert.False(t, managesAdmissionPolicies(ctx, tftypes.MapNull(tftypes.StringType)))
	assert.False(t, managesAdmissionPolicies(ctx, resourcesMap(t, testAppManifest)), "applications alone leave portal policies alone")
	assert.True(t, managesAdmissionPolicies(ctx, resourcesMap(t, testAppManifest, testPolicyManifest)))
	assert.True(t, managesAdmissionPolicies(withManagedAdmissionPolicies(ctx, true), resourcesMap(t, testAppManifest)),
		"removing the last policy still replaces the stored set")
	assert.False(t, managesAdmissionPolicies(ctx, resourcesMap(t, `not json`)))
}

func TestBuildApplyRequest_AdmissionPolicyPrune(t *testing.T) {
	ctx := context.Background()
	has := func(types []argocdv1.PruneResourceType) bool {
		return slices.Contains(types, argocdv1.PruneResourceType_PRUNE_RESOURCE_TYPE_VALIDATING_ADMISSION_POLICIES)
	}
	var diags diag.Diagnostics

	withPolicy := &types.Instance{ArgoCDResources: resourcesMap(t, testPolicyManifest)}
	assert.True(t, has(buildApplyRequest(ctx, &diags, withPolicy, "org", "ws").GetPruneResourceTypes()))

	withoutPolicy := &types.Instance{ArgoCDResources: resourcesMap(t, testAppManifest)}
	assert.False(t, has(buildApplyRequest(ctx, &diags, withoutPolicy, "org", "ws").GetPruneResourceTypes()))
}

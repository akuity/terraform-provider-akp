package types

import (
	"testing"

	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The acceptance tests can only assert shard == "us0", because the server reports
// every default-shard instance by the default shard's display name. That assertion
// passes even if the provider never transmitted the field, so it cannot catch a
// placement regression on its own (see the TODOs in resource_akp_instance_test.go
// and resource_akp_kargo_test.go).
//
// These tests close that gap from the other side: they pin that a configured shard
// actually survives the TF -> API map conversion that buildArgoCD/buildKargo feed
// into the apply request. TFToMapWithOverrides walks the tfsdk struct tags with
// reflection and never marshals the v1alpha1 mirror, so the `omitempty` on
// v1alpha1.{ArgoCD,Kargo}Spec.Shard governs exported manifests, NOT what the
// provider transmits: here a null shard is dropped and a known empty string is
// sent as "". Both products must agree on all of it, which is what these assert.

func argoCDSpecMap(t *testing.T, argocd *ArgoCD) map[string]any {
	t.Helper()
	raw := TFToMapWithOverrides(argocd, OverridesMap, nil)
	require.NotNil(t, raw, "TFToMapWithOverrides returned nil")
	spec, ok := raw["spec"].(map[string]any)
	require.True(t, ok, "converted map has no spec object: %#v", raw)
	return spec
}

func kargoSpecMap(t *testing.T, kargo *Kargo) map[string]any {
	t.Helper()
	raw := TFToMapWithOverrides(kargo, KargoOverridesMap, KargoRenamesMap)
	require.NotNil(t, raw, "TFToMapWithOverrides returned nil")
	spec, ok := raw["spec"].(map[string]any)
	require.True(t, ok, "converted map has no spec object: %#v", raw)
	return spec
}

func TestArgoCDShardReachesAPIRequest(t *testing.T) {
	testCases := map[string]struct {
		shard     tftypes.String
		wantKey   bool
		wantValue string
	}{
		"configured non-default shard is transmitted": {
			shard: tftypes.StringValue("us1"), wantKey: true, wantValue: "us1",
		},
		"configured default shard name is transmitted verbatim": {
			// The server, not the provider, maps the display name back to the
			// internal empty string.
			shard: tftypes.StringValue("us0"), wantKey: true, wantValue: "us0",
		},
		"null shard is omitted": {
			shard: tftypes.StringNull(), wantKey: false,
		},
		"explicit empty shard is sent through as empty": {
			// Only reachable by writing `shard = ""`; an omitted attribute is null and
			// drops out above. The server reads "" as the default shard, and Kargo
			// behaves identically here.
			shard: tftypes.StringValue(""), wantKey: true, wantValue: "",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			spec := argoCDSpecMap(t, &ArgoCD{
				Spec: ArgoCDSpec{
					Description: tftypes.StringValue("test"),
					Version:     tftypes.StringValue("v3.4.4"),
					Shard:       tc.shard,
				},
			})
			got, ok := spec["shard"]
			if !tc.wantKey {
				assert.False(t, ok, "expected shard to be omitted, got %#v", got)
				return
			}
			require.True(t, ok, "shard missing from converted spec: %#v", spec)
			assert.Equal(t, tc.wantValue, got)
		})
	}
}

func TestKargoShardReachesAPIRequest(t *testing.T) {
	testCases := map[string]struct {
		shard     tftypes.String
		wantKey   bool
		wantValue string
	}{
		"configured non-default shard is transmitted": {
			shard: tftypes.StringValue("us1"), wantKey: true, wantValue: "us1",
		},
		"configured default shard name is transmitted verbatim": {
			shard: tftypes.StringValue("us0"), wantKey: true, wantValue: "us0",
		},
		"null shard is omitted": {
			shard: tftypes.StringNull(), wantKey: false,
		},
		"explicit empty shard is sent through as empty": {
			// Identical to the Argo CD behaviour above — the two must not diverge.
			shard: tftypes.StringValue(""), wantKey: true, wantValue: "",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			spec := kargoSpecMap(t, &Kargo{
				Spec: KargoSpec{
					Description: tftypes.StringValue("test"),
					Version:     tftypes.StringValue("v1.11.0-ak.0"),
					Shard:       tc.shard,
				},
			})
			got, ok := spec["shard"]
			if !tc.wantKey {
				assert.False(t, ok, "expected shard to be omitted, got %#v", got)
				return
			}
			require.True(t, ok, "shard missing from converted spec: %#v", spec)
			assert.Equal(t, tc.wantValue, got)
		})
	}
}

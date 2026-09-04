package types

import (
	"testing"

	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestKargoIPAllowListReachesAPIRequest(t *testing.T) {
	testCases := map[string]struct {
		list []*KargoIPAllowListEntry
		want any
	}{
		"removed list is sent as empty so the merge patch clears it": {
			list: nil,
			want: []any{},
		},
		"explicit empty list is sent as empty": {
			list: []*KargoIPAllowListEntry{},
			want: []any{},
		},
		"entries are transmitted": {
			list: []*KargoIPAllowListEntry{{
				Ip:          tftypes.StringValue("88.88.88.88"),
				Description: tftypes.StringValue("office"),
			}},
			want: []any{map[string]any{"ip": "88.88.88.88", "description": "office"}},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			spec := kargoSpecMap(t, &Kargo{Spec: KargoSpec{
				KargoInstanceSpec: KargoInstanceSpec{IpAllowList: tc.list},
			}})
			instanceSpec, ok := spec["kargoInstanceSpec"].(map[string]any)
			require.True(t, ok, "converted spec has no kargoInstanceSpec object: %#v", spec)
			got, ok := instanceSpec["ipAllowList"]
			require.True(t, ok, "ipAllowList missing from kargoInstanceSpec: %#v", instanceSpec)
			require.Equal(t, tc.want, got)
		})
	}
}

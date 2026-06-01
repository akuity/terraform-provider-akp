package akp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStripRoleNamespace(t *testing.T) {
	in := []string{"organization/member", "workspace/admin", "viewer", "", "weird/"}
	got := stripRoleNamespace(in)
	require.Equal(t, []string{"member", "admin", "viewer", "", "weird/"}, got)
	require.Nil(t, stripRoleNamespace(nil))
	require.Nil(t, stripRoleNamespace([]string{}))
}

func TestFilterRolesByNamespace(t *testing.T) {
	in := []string{"organization/member", "workspace/admin", "workspace/member", "loose", "organization/owner"}

	t.Run("workspace scope drops auto-appended org/member", func(t *testing.T) {
		require.Equal(t,
			[]string{"workspace/admin", "workspace/member", "loose"},
			filterRolesByNamespace(in, "workspace"),
		)
	})
	t.Run("organization scope drops workspace roles", func(t *testing.T) {
		require.Equal(t,
			[]string{"organization/member", "loose", "organization/owner"},
			filterRolesByNamespace(in, "organization"),
		)
	})
	t.Run("empty input", func(t *testing.T) {
		require.Nil(t, filterRolesByNamespace(nil, "workspace"))
		require.Nil(t, filterRolesByNamespace([]string{}, "workspace"))
	})
}

func TestExpireInDurationRegex(t *testing.T) {
	t.Run("accepts", func(t *testing.T) {
		valid := []string{
			"1h", "30m", "8760h", "30d", "500ms", "1h30m", "1.5h", "10s", "5us", "5µs", "5ns",
		}
		for _, v := range valid {
			require.True(t, expireInDurationRegex.MatchString(v), "expected %q to match", v)
		}
	})
	t.Run("rejects", func(t *testing.T) {
		invalid := []string{
			"", " ", "abc", "30", "1y", "h", "1h ", " 1h", "1hour", "-1h", "1h-30m",
		}
		for _, v := range invalid {
			require.False(t, expireInDurationRegex.MatchString(v), "expected %q to be rejected", v)
		}
	})
}

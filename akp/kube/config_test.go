//go:build !acc

package kube

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	for in, want := range map[string]string{
		"~":              home,
		"~/.kube/config": filepath.Join(home, "/.kube/config"),
		`~\.kube\config`: filepath.Join(home, `\.kube\config`),
		"/etc/kube":      "/etc/kube",
		"~user/x":        "~user/x",
	} {
		got, err := expandHome(in)
		require.NoError(t, err)
		require.Equal(t, want, got, in)
	}
}

func TestInitializeConfigurationConfigPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	const clusterConfig = `apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: https://kubeconfig.example.test
users:
- name: test
  user:
    token: kubeconfig-test-token
`
	const contextConfig = `contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
`
	configPath := filepath.Join(home, "config")
	clusterPath := filepath.Join(home, "cluster")
	require.NoError(t, os.WriteFile(configPath, []byte(clusterConfig+contextConfig), 0o600))
	require.NoError(t, os.WriteFile(clusterPath, []byte(clusterConfig), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(home, "context"), []byte("apiVersion: v1\nkind: Config\n"+contextConfig), 0o600))

	for name, paths := range map[string][]string{
		"absolute path":  {configPath},
		"home path":      {"~/config"},
		"merged configs": {clusterPath, "~/context"},
	} {
		t.Run(name, func(t *testing.T) {
			configPaths, diags := tftypes.ListValueFrom(context.Background(), tftypes.StringType, paths)
			require.False(t, diags.HasError(), "%v", diags)
			cfg, err := InitializeConfiguration(context.Background(), &types.Kubeconfig{ConfigPaths: configPaths})
			require.NoError(t, err)
			require.Equal(t, "https://kubeconfig.example.test", cfg.Host)
			require.Equal(t, "kubeconfig-test-token", cfg.BearerToken)
		})
	}
}

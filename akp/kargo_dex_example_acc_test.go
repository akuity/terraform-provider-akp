//go:build !unit

package akp

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
)

func runKargoDexExample(t *testing.T) {
	example, err := os.ReadFile("../examples/resources/akp_kargo_instance/resource_dex.tf")
	require.NoError(t, err)
	// CI has no cert-manager. Exercise the Dex example without provisioning a
	// custom-domain certificate; the hostname still supplies the OAuth callback.
	const fqdnAttribute = "fqdn    = var.kargo_hostname"
	require.Contains(t, string(example), fqdnAttribute)
	exampleConfig := strings.Replace(string(example), fqdnAttribute, "", 1)
	name := acctest.RandomWithPrefix("kargo-dex")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: providerConfig + exampleConfig,
			ConfigVariables: config.Variables{
				"kargo_name":           config.StringVariable(name),
				"kargo_version":        config.StringVariable(getKargoVersion()),
				"kargo_hostname":       config.StringVariable(name + ".example.com"),
				"github_client_id":     config.StringVariable("example-client-id"),
				"github_client_secret": config.StringVariable("example-client-secret"),
				"github_org":           config.StringVariable("example-org"),
			},
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("akp_kargo_instance.github_sso", "name", name),
				resource.TestCheckNoResourceAttr("akp_kargo_instance.github_sso", "kargo.spec.fqdn"),
				resource.TestCheckResourceAttr("akp_kargo_instance.github_sso", "kargo.spec.oidc_config.dex_enabled", "true"),
				resource.TestCheckResourceAttr("akp_kargo_instance.github_sso", "kargo.spec.oidc_config.dex_config_secret.GITHUB_CLIENT_SECRET", "example-client-secret"),
				resource.TestCheckTypeSetElemAttr("akp_kargo_instance.github_sso", "kargo.spec.oidc_config.admin_account.claims.groups.values.*", "example-org:platform-admins"),
				resource.TestCheckTypeSetElemAttr("akp_kargo_instance.github_sso", "kargo.spec.oidc_config.viewer_account.claims.groups.values.*", "example-org:developers"),
			),
		}},
	})
}

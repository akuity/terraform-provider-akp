//go:build !unit

package akp

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccInstanceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + testAccInstanceDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.akp_instance.test", "id", "6pzhawvy4echbd8x"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "name", "test-cluster"),

					// argocd
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.description", "This is used by the terraform provider to test managing clusters."),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.version", "v2.8.0"),
					// argocd.instance_spec
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.subdomain", "6pzhawvy4echbd8x"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.declarative_management_enabled", "false"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.image_updater_enabled", "false"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.appset_policy.policy", "sync"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.appset_policy.override_policy", "false"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.host_aliases.#", "1"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.host_aliases.0.ip", "1.2.3.4"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.host_aliases.0.hostnames.#", "2"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.host_aliases.0.hostnames.0", "test-1"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.host_aliases.0.hostnames.1", "test-2"),

					// argocd_cm
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd_cm.%", "9"),

					// config_management_plugins
					resource.TestCheckResourceAttr("data.akp_instance.test", "config_management_plugins.%", "2"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "config_management_plugins.helmfile.enabled", "false"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "config_management_plugins.tanka.enabled", "true"),
				),
			},
		},
	})
}

const testAccInstanceDataSourceConfig = `
data "akp_instance" "test" {
	name = "test-cluster"
}
`

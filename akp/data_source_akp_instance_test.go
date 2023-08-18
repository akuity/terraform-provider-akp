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
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.version", "v2.7.9"),
					// argocd.instance_spec
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.subdomain", "6pzhawvy4echbd8x"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.declarative_management_enabled", "false"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.image_updater_enabled", "false"),

					// argocd_cm
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd_cm.data.%", "3"),
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

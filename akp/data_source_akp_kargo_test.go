package akp

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKargoDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.akp_kargo.test", "id", "5gjcg0rh8fjemhc0"),
					resource.TestCheckResourceAttr("data.akp_kargo.test", "name", "kargo"),
					resource.TestCheckResourceAttr("data.akp_kargo.test", "version", "v1.1.1"),
					resource.TestCheckResourceAttr("data.akp_kargo.test", "workspace_id", "sw3lpl9tr4iuaj8z"),
					// spec
					resource.TestCheckResourceAttr("data.akp_kargo.test", "spec.backend_ip_allow_list_enabled", "false"),
					resource.TestCheckResourceAttr("data.akp_kargo.test", "spec.ip_allow_list.#", "0"),
					resource.TestCheckResourceAttr("data.akp_kargo.test", "spec.global_credentials_ns.#", "0"),
					resource.TestCheckResourceAttr("data.akp_kargo.test", "spec.global_service_account_ns.#", "0"),
					resource.TestCheckResourceAttr("data.akp_kargo.test", "spec.default_shard_agent", ""),
					// agent customization defaults
					resource.TestCheckResourceAttr("data.akp_kargo.test", "spec.agent_customization_defaults.auto_upgrade_disabled", "false"),
					resource.TestCheckResourceAttr("data.akp_kargo.test", "spec.agent_customization_defaults.kustomization", ""),
				),
			},
		},
	})
}

const testAccKargoDataSourceConfig = `
data "akp_kargo" "test" {
  name = "test-instance"
}
`

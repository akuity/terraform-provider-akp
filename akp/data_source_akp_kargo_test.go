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
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "id", "5gjcg0rh8fjemhc0"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "name", "test-instance"),
					// spec
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance.kargo.spec", "version", "v1.1.1"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance.kargo.spec", "kargo_instance_spec.ip_allow_list.#", "0"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance.kargo.spec", "kargo_instance_spec.global_credentials_ns.#", "2"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance.kargo.spec", "kargo_instance_spec.global_service_account_ns.#", "1"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance.kargo.spec", "kargo_instance_spec.default_shard_agent", "kgbgel4pst55klf9"),
					// cm
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance.kargo.kargo_cm", "adminAccountEnabled", "true"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance.kargo.kargo_cm", "adminAccountTokenTtl", "24h"),
				),
			},
		},
	})
}

const testAccKargoDataSourceConfig = `
data "akp_kargo_instance" "test-instance" {
  name = "test-instance"
}
`

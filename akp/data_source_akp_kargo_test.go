package akp

import (
	"fmt"
	"strings"
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
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "kargo.spec.version", "v1.2.2"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "kargo.spec.kargo_instance_spec.ip_allow_list.#", "0"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "kargo.spec.kargo_instance_spec.global_credentials_ns.#", "2"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "kargo.spec.kargo_instance_spec.global_service_account_ns.#", "1"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "kargo.spec.kargo_instance_spec.default_shard_agent", "kgbgel4pst55klf9"),
					// cm
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "kargo_cm.%", "2"),

					// Test Kargo Resources
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "kargo_resources.%", "12"),

					resource.TestCheckResourceAttrWith("data.akp_kargo_instance.test-instance", "kargo_resources.kargo.akuity.io/v1alpha1/Project//kargo-demo", func(value string) error {
						if !strings.Contains(value, "kargo-demo") {
							return fmt.Errorf("expected to contain name: %s", value)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("data.akp_kargo_instance.test-instance", "kargo_resources.kargo.akuity.io/v1alpha1/Warehouse/kargo-demo/kargo-demo", func(value string) error {
						if !strings.Contains(value, "public.ecr.aws/nginx/nginx") {
							return fmt.Errorf("expected to contain name: %s", value)
						}
						return nil
					}),
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

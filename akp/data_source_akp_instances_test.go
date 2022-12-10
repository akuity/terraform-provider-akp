package akp

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccInstancesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + testAccInstancesDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.akp_instances.test", "instances.#", "1"),
					resource.TestCheckResourceAttr("data.akp_instances.test", "instances.0.id", "gnjajx9dkszyyp55"),
					resource.TestCheckResourceAttr("data.akp_instances.test", "instances.0.name", "existing-instance"),
					resource.TestCheckResourceAttr("data.akp_instances.test", "instances.0.description", "Test description"),
					resource.TestCheckResourceAttr("data.akp_instances.test", "instances.0.hostname", "gnjajx9dkszyyp55.cd.akuity.cloud"),
					resource.TestCheckResourceAttr("data.akp_instances.test", "instances.0.version", "v2.5.3"),
				),
			},
		},
	})
}

const testAccInstancesDataSourceConfig = `
data "akp_instances" "test" {}
`

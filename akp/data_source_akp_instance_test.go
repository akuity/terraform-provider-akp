package akp

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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
					resource.TestCheckResourceAttr("data.akp_instance.test", "name", "existing-instance"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "id", "gnjajx9dkszyyp55"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "description", "Test description"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "hostname", "gnjajx9dkszyyp55.cd.akuity.cloud"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "version", "v2.5.3"),
				),
			},
		},
	})
}

const testAccInstanceDataSourceConfig = `
data "akp_instance" "test" {
	name = "existing-instance"
}
`

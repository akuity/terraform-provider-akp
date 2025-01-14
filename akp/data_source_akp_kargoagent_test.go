//go:build !unit

package akp

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKargoAgentDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "id", "kgbgel4pst55klf9"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "instance_id", "5gjcg0rh8fjemhc0"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "name", "test-agent"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "spec.data.namespace", "akuity"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "spec.data.labels.app", "test"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "spec.data.annotations.app", "test"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "spec.data.target_version", "0.5.52"),
				),
			},
		},
	})
}

const testAccKargoAgentDataSourceConfig = `
data "akp_kargo_agent" "test" {
  instance_id = "5gjcg0rh8fjemhc0"
  name = "test-agent"
}
`

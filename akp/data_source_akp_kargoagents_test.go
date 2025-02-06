//go:build !unit

package akp

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccKargoAgentsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentsDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKargoAgentAttributes("data.akp_kargo_agents.test-agents", "test-agent"),
				),
			},
		},
	})
}

const testAccKargoAgentsDataSourceConfig = `
data "akp_kargo_agents" "test-agents" {
  instance_id = "5gjcg0rh8fjemhc0"
}
`

func testAccCheckKargoAgentAttributes(dataSourceName string, targetKargoAgentName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("data source %s not found", dataSourceName)
		}
		clusters := rs.Primary.Attributes
		for i := 0; ; i++ {
			if clusters[fmt.Sprintf("kargo_agents.%d.name", i)] == "" {
				break
			}
			if clusters[fmt.Sprintf("kargo_agents.%d.name", i)] == targetKargoAgentName {
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("kargo_agents.%d.instance_id", i), "5gjcg0rh8fjemhc0")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("kargo_agents.%d.id", i), "kgbgel4pst55klf9")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("kargo_agents.%d.name", i), "test-agent")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("kargo_agents.%d.namespace", i), "akuity")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("kargo_agents.%d.labels.app", i), "test")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("kargo_agents.%d.annotations.app", i), "test")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("kargo_agents.%d.spec.data.size", i), "small")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("kargo_agents.%d.spec.data.target_version", i), "0.5.54")(s); err != nil {
					return err
				}
				return nil
			}
		}
		return fmt.Errorf("target kargo agent %s not found", targetKargoAgentName)
	}
}

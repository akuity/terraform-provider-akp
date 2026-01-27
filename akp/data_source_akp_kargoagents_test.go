//go:build !unit

package akp

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccKargoAgentsDataSource(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("ds-kargo-agents-%s", acctest.RandString(8))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + getTestAccKargoAgentsDataSourceConfig(getKargoInstanceId(), name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKargoAgentAttributes("data.akp_kargo_agents.test-agents", name),
				),
			},
		},
	})
}

func getTestAccKargoAgentsDataSourceConfig(instanceId, name string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "akuity"
  labels = {
    app = "test"
  }
  annotations = {
    app = "test"
  }
  spec = {
    description = "test kargo agent for data source"
    data = {
      size                  = "small"
      auto_upgrade_disabled = false
      remote_argocd         = %q
      akuity_managed        = false
    }
  }
  remove_agent_resources_on_destroy = true
}

data "akp_kargo_agents" "test-agents" {
  instance_id = akp_kargo_agent.test.instance_id
}
`, instanceId, name, getInstanceId())
}

func testAccCheckKargoAgentAttributes(dataSourceName, targetKargoAgentName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("data source %s not found", dataSourceName)
		}
		clusters := rs.Primary.Attributes
		for i := 0; ; i++ {
			if clusters[fmt.Sprintf("agents.%d.name", i)] == "" {
				break
			}
			if clusters[fmt.Sprintf("agents.%d.name", i)] == targetKargoAgentName {
				if err := resource.TestCheckResourceAttrSet(dataSourceName, fmt.Sprintf("agents.%d.instance_id", i))(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttrSet(dataSourceName, fmt.Sprintf("agents.%d.id", i))(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("agents.%d.name", i), targetKargoAgentName)(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("agents.%d.namespace", i), "akuity")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("agents.%d.labels.app", i), "test")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("agents.%d.annotations.app", i), "test")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("agents.%d.spec.data.size", i), "small")(s); err != nil {
					return err
				}
				return nil
			}
		}
		return fmt.Errorf("target kargo agent %s not found", targetKargoAgentName)
	}
}

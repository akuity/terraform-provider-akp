//go:build !unit

package akp

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKargoAgentDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + getAccKargoAgentDataSourceConfig(getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "instance_id", getKargoInstanceId()),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "name", "test-agent"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "namespace", "akuity"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "labels.app", "test"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "annotations.app", "test"),
				),
			},
		},
	})
}

func getAccKargoAgentDataSourceConfig(instanceId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = "test-agent"
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

data "akp_kargo_agent" "test" {
  name        = akp_kargo_agent.test.name
  instance_id = akp_kargo_agent.test.instance_id
}
`, instanceId, getInstanceId())
}

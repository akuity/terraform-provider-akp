//go:build !unit

package akp

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
)

func TestAccKargoDefaultShardAgentResource(t *testing.T) {
	t.Parallel()
	agentName := fmt.Sprintf("kargoagent-default-%s", acctest.RandString(8))

	t.Cleanup(func() {
		_ = clearDefaultShardAgent(getKargoInstanceId())
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create agent and set as default shard agent
			{
				Config: providerConfig + testAccKargoDefaultShardAgentResourceConfig(agentName, getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_default_shard_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_default_shard_agent.test", "kargo_instance_id", getKargoInstanceId()),
					resource.TestCheckResourceAttrSet("akp_kargo_default_shard_agent.test", "agent_id"),
					testAccCheckDefaultShardAgentIsSet("akp_kargo_default_shard_agent.test"),
				),
			},
			// ImportState testing
			{
				ResourceName: "akp_kargo_default_shard_agent.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["akp_kargo_default_shard_agent.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					instanceID := rs.Primary.Attributes["kargo_instance_id"]
					agentID := rs.Primary.Attributes["agent_id"]
					return fmt.Sprintf("%s/%s", instanceID, agentID), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckDefaultShardAgentIsSet(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		instanceID := rs.Primary.Attributes["kargo_instance_id"]
		agentID := rs.Primary.Attributes["agent_id"]

		akpCli := getTestAkpCli()
		ctx := context.Background()
		ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())

		instancesResp, err := akpCli.KargoCli.ListKargoInstances(ctx, &kargov1.ListKargoInstancesRequest{
			OrganizationId: akpCli.OrgId,
		})
		if err != nil {
			return fmt.Errorf("failed to list kargo instances: %v", err)
		}

		for _, instance := range instancesResp.GetInstances() {
			if instance.GetId() == instanceID {
				currentDefault := instance.GetSpec().GetDefaultShardAgent()
				if currentDefault != agentID {
					return fmt.Errorf("expected default shard agent %q, got %q", agentID, currentDefault)
				}
				return nil
			}
		}

		return fmt.Errorf("kargo instance %s not found", instanceID)
	}
}

func testAccKargoDefaultShardAgentResourceConfig(agentName, kargoInstanceId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "default" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "Default shard agent test"
    data = {
      size           = "small"
      remote_argocd  = %q
      akuity_managed = false
    }
  }
  remove_agent_resources_on_destroy = true
}

resource "akp_kargo_default_shard_agent" "test" {
  kargo_instance_id = %q
  agent_id          = akp_kargo_agent.default.id
}
`, kargoInstanceId, agentName, getInstanceId(), kargoInstanceId)
}

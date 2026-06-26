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
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
)

const teamResourceName = "akp_team.test"

// runTeamResource covers create, in-place description update, import, and
// delete. The team name is the natural key, so it is stable across the update.
func runTeamResource(t *testing.T) {
	suffix := acctest.RandString(8)
	name := fmt.Sprintf("tf-acc-team-%s", suffix)
	descInitial := "tf acceptance team"
	descUpdated := "tf acceptance team (updated)"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTeamDestroyed,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccTeamConfig(name, descInitial),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(teamResourceName, "name", name),
					resource.TestCheckResourceAttr(teamResourceName, "description", descInitial),
					resource.TestCheckResourceAttrSet(teamResourceName, "create_time"),
					resource.TestCheckResourceAttr(teamResourceName, "member_count", "0"),
					testAccCheckTeamExists(teamResourceName),
				),
			},
			{
				ResourceName:      teamResourceName,
				ImportState:       true,
				ImportStateId:     name,
				ImportStateVerify: true,
				// Teams have no `id`; the natural key is `name`.
				ImportStateVerifyIdentifierAttribute: "name",
			},
			{
				Config: providerConfig + testAccTeamConfig(name, descUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(teamResourceName, "name", name),
					resource.TestCheckResourceAttr(teamResourceName, "description", descUpdated),
					testAccCheckTeamExists(teamResourceName),
				),
			},
		},
	})
}

func testAccCheckTeamExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		teamName := rs.Primary.Attributes["name"]
		if teamName == "" {
			return fmt.Errorf("no name set for %s", name)
		}
		cli := getTestAkpCli()
		if cli == nil {
			return fmt.Errorf("could not get test client")
		}
		ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
		resp, err := cli.OrgCli.GetTeam(ctx, &orgcv1.GetTeamRequest{
			OrganizationId: cli.OrgId,
			Name:           teamName,
		})
		if err != nil {
			return fmt.Errorf("GetTeam(%s) failed: %w", teamName, err)
		}
		if resp.GetUserTeam() == nil || resp.GetUserTeam().GetTeam().GetName() != teamName {
			return fmt.Errorf("GetTeam(%s) returned mismatched payload", teamName)
		}
		return nil
	}
}

func testAccCheckTeamDestroyed(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "akp_team" {
			continue
		}
		teamName := rs.Primary.Attributes["name"]
		if teamName == "" {
			continue
		}
		cli := getTestAkpCli()
		if cli == nil {
			return fmt.Errorf("could not get test client")
		}
		ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
		_, err := cli.OrgCli.GetTeam(ctx, &orgcv1.GetTeamRequest{
			OrganizationId: cli.OrgId,
			Name:           teamName,
		})
		if err == nil {
			return fmt.Errorf("team %s still exists after destroy", teamName)
		}
		if !isGoneErr(err) {
			return fmt.Errorf("GetTeam after destroy returned unexpected error: %w", err)
		}
	}
	return nil
}

func testAccTeamConfig(name, description string) string {
	return fmt.Sprintf(`
resource "akp_team" "test" {
  name        = %q
  description = %q
}
`, name, description)
}

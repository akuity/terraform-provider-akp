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

const workspaceMemberResourceName = "akp_workspace_member.test"

// runWorkspaceMemberResource covers create, in-place role update, import, and
// delete for a team-backed workspace member. The team and workspace are
// provisioned via their own resources in the same config, exercising the full
// akp_team -> akp_workspace_member chain. id is stable across the role update
// because role is mutable server-side.
func runWorkspaceMemberResource(t *testing.T) {
	suffix := acctest.RandString(8)
	workspaceName := fmt.Sprintf("tf-acc-ws-%s", suffix)
	teamName := fmt.Sprintf("tf-acc-team-%s", suffix)

	var initialID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckWorkspaceMemberDestroyed,
			testAccCheckWorkspaceDestroyed,
			testAccCheckTeamDestroyed,
		),
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccWorkspaceMemberConfigTeam(workspaceName, teamName, "member"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(workspaceMemberResourceName, "id"),
					resource.TestCheckResourceAttrSet(workspaceMemberResourceName, "workspace_id"),
					resource.TestCheckResourceAttr(workspaceMemberResourceName, "workspace", workspaceName),
					resource.TestCheckResourceAttr(workspaceMemberResourceName, "team_name", teamName),
					resource.TestCheckResourceAttr(workspaceMemberResourceName, "role", "member"),
					testAccCheckWorkspaceMemberExists(workspaceMemberResourceName),
					captureWorkspaceMemberID(workspaceMemberResourceName, &initialID),
				),
			},
			{
				ResourceName:      workspaceMemberResourceName,
				ImportState:       true,
				ImportStateIdFunc: workspaceMemberImportID(workspaceMemberResourceName),
				ImportStateVerify: true,
				// On import the member identity is recovered as team_name, which
				// matches config; nothing extra to ignore.
			},
			{
				Config: providerConfig + testAccWorkspaceMemberConfigTeam(workspaceName, teamName, "admin"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(workspaceMemberResourceName, "role", "admin"),
					expectWorkspaceMemberSameID(workspaceMemberResourceName, &initialID),
					testAccCheckWorkspaceMemberExists(workspaceMemberResourceName),
				),
			},
		},
	})
}

func testAccCheckWorkspaceMemberExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID set for %s", name)
		}
		cli := getTestAkpCli()
		if cli == nil {
			return fmt.Errorf("could not get test client")
		}
		ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
		resp, err := cli.OrgCli.GetWorkspaceMember(ctx, &orgcv1.GetWorkspaceMemberRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    rs.Primary.Attributes["workspace_id"],
			Id:             rs.Primary.ID,
		})
		if err != nil {
			return fmt.Errorf("GetWorkspaceMember(%s) failed: %w", rs.Primary.ID, err)
		}
		if resp.GetWorkspaceMember() == nil || resp.GetWorkspaceMember().GetId() != rs.Primary.ID {
			return fmt.Errorf("GetWorkspaceMember(%s) returned mismatched payload", rs.Primary.ID)
		}
		return nil
	}
}

func testAccCheckWorkspaceMemberDestroyed(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "akp_workspace_member" {
			continue
		}
		if rs.Primary.ID == "" {
			continue
		}
		cli := getTestAkpCli()
		if cli == nil {
			return fmt.Errorf("could not get test client")
		}
		ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
		_, err := cli.OrgCli.GetWorkspaceMember(ctx, &orgcv1.GetWorkspaceMemberRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    rs.Primary.Attributes["workspace_id"],
			Id:             rs.Primary.ID,
		})
		if err == nil {
			return fmt.Errorf("workspace member %s still exists after destroy", rs.Primary.ID)
		}
		if !isGoneErr(err) {
			return fmt.Errorf("GetWorkspaceMember after destroy returned unexpected error: %w", err)
		}
	}
	return nil
}

func captureWorkspaceMemberID(name string, id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		*id = rs.Primary.ID
		if *id == "" {
			return fmt.Errorf("expected id to be set")
		}
		return nil
	}
}

func expectWorkspaceMemberSameID(name string, prev *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		if rs.Primary.ID != *prev {
			return fmt.Errorf("expected id to remain %s across in-place role update; got %s", *prev, rs.Primary.ID)
		}
		return nil
	}
}

// workspaceMemberImportID builds the <workspace_name>/<member_id> import ID
// from the resource's terraform state.
func workspaceMemberImportID(name string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return "", fmt.Errorf("not found: %s", name)
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["workspace"], rs.Primary.ID), nil
	}
}

func testAccWorkspaceMemberConfigTeam(workspaceName, teamName, role string) string {
	return fmt.Sprintf(`
resource "akp_workspace" "test" {
  name = %q
}

resource "akp_team" "test" {
  name = %q
}

resource "akp_workspace_member" "test" {
  workspace = akp_workspace.test.name
  team_name = akp_team.test.name
  role      = %q
}
`, workspaceName, teamName, role)
}

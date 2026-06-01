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

const workspaceResourceName = "akp_workspace.test"

// runWorkspaceResource covers create, in-place update (name + description),
// import, and delete. id is stable across the update step because both fields
// are mutable server-side.
func runWorkspaceResource(t *testing.T) {
	suffix := acctest.RandString(8)
	name := fmt.Sprintf("tf-acc-%s", suffix)
	nameUpdated := fmt.Sprintf("tf-acc-%s-upd", suffix)
	descInitial := "tf acceptance workspace"
	descUpdated := "tf acceptance workspace (updated)"

	var initialID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWorkspaceDestroyed,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccWorkspaceConfig(name, descInitial),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(workspaceResourceName, "id"),
					resource.TestCheckResourceAttr(workspaceResourceName, "name", name),
					resource.TestCheckResourceAttr(workspaceResourceName, "description", descInitial),
					resource.TestCheckResourceAttrSet(workspaceResourceName, "create_time"),
					resource.TestCheckResourceAttr(workspaceResourceName, "is_default", "false"),
					testAccCheckWorkspaceExists(workspaceResourceName),
					captureWorkspaceID(workspaceResourceName, &initialID),
				),
			},
			{
				ResourceName:      workspaceResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: providerConfig + testAccWorkspaceConfig(nameUpdated, descUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(workspaceResourceName, "name", nameUpdated),
					resource.TestCheckResourceAttr(workspaceResourceName, "description", descUpdated),
					expectWorkspaceSameID(workspaceResourceName, &initialID),
					testAccCheckWorkspaceExists(workspaceResourceName),
				),
			},
		},
	})
}

func testAccCheckWorkspaceExists(name string) resource.TestCheckFunc {
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
		resp, err := cli.OrgCli.GetWorkspace(ctx, &orgcv1.GetWorkspaceRequest{
			OrganizationId: cli.OrgId,
			Id:             rs.Primary.ID,
		})
		if err != nil {
			return fmt.Errorf("GetWorkspace(%s) failed: %w", rs.Primary.ID, err)
		}
		if resp.GetWorkspace() == nil || resp.GetWorkspace().GetId() != rs.Primary.ID {
			return fmt.Errorf("GetWorkspace(%s) returned mismatched payload", rs.Primary.ID)
		}
		return nil
	}
}

func testAccCheckWorkspaceDestroyed(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "akp_workspace" {
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
		_, err := cli.OrgCli.GetWorkspace(ctx, &orgcv1.GetWorkspaceRequest{
			OrganizationId: cli.OrgId,
			Id:             rs.Primary.ID,
		})
		if err == nil {
			return fmt.Errorf("workspace %s still exists after destroy", rs.Primary.ID)
		}
		if !isGoneErr(err) {
			return fmt.Errorf("GetWorkspace after destroy returned unexpected error: %w", err)
		}
	}
	return nil
}

func captureWorkspaceID(name string, id *string) resource.TestCheckFunc {
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

func expectWorkspaceSameID(name string, prev *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		if rs.Primary.ID != *prev {
			return fmt.Errorf("expected id to remain %s across in-place update; got %s", *prev, rs.Primary.ID)
		}
		return nil
	}
}

func testAccWorkspaceConfig(name, description string) string {
	return fmt.Sprintf(`
resource "akp_workspace" "test" {
  name        = %q
  description = %q
}
`, name, description)
}

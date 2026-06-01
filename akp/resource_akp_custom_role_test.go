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

const customRoleResourceName = "akp_custom_role.test"

// runCustomRoleResource exercises create / read / in-place update / import /
// delete for an org-scoped custom role. The id must remain stable across the
// update step — these fields are NOT RequiresReplace.
func runCustomRoleResource(t *testing.T) {
	name := fmt.Sprintf("tf-acc-%s", acctest.RandString(8))
	descInitial := "initial"
	descUpdated := "updated"
	policyInitial := `p, role:tf-acc, organization/apikeys, get, *`
	policyUpdated := `p, role:tf-acc, organization/apikeys, create, *`

	var initialID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCustomRoleDestroyed,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccCustomRoleConfigOrg(name, descInitial, policyInitial),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(customRoleResourceName, "id"),
					resource.TestCheckResourceAttr(customRoleResourceName, "name", name),
					resource.TestCheckResourceAttr(customRoleResourceName, "description", descInitial),
					resource.TestCheckResourceAttr(customRoleResourceName, "policy", policyInitial),
					testAccCheckCustomRoleExists(customRoleResourceName),
					captureCustomRoleID(customRoleResourceName, &initialID),
				),
			},
			testAccCustomRoleImportStateStep(),
			{
				// Description and policy can be edited in place; id must NOT change.
				Config: providerConfig + testAccCustomRoleConfigOrg(name, descUpdated, policyUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(customRoleResourceName, "description", descUpdated),
					resource.TestCheckResourceAttr(customRoleResourceName, "policy", policyUpdated),
					expectCustomRoleSameID(customRoleResourceName, &initialID),
					testAccCheckCustomRoleExists(customRoleResourceName),
				),
			},
		},
	})
}

// runCustomRoleResourceWorkspace covers the workspace-scoped path. The config
// provisions an akp_workspace inline so the test is self-contained and runs
// as part of TestAccAll without extra setup.
func runCustomRoleResourceWorkspace(t *testing.T) {
	suffix := acctest.RandString(8)
	workspaceName := fmt.Sprintf("tf-acc-ws-%s", suffix)
	name := fmt.Sprintf("tf-acc-ws-role-%s", suffix)
	policy := `p, role:tf-acc-ws, workspace/instances, get, *`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckCustomRoleDestroyed,
			testAccCheckWorkspaceDestroyed,
		),
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccCustomRoleConfigWorkspaceInline(workspaceName, name, "ws-scoped", policy),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(customRoleResourceName, "id"),
					resource.TestCheckResourceAttr(customRoleResourceName, "workspace", workspaceName),
					resource.TestCheckResourceAttr(customRoleResourceName, "name", name),
					testAccCheckCustomRoleExistsWorkspace(customRoleResourceName, workspaceName),
				),
			},
			testAccCustomRoleImportStateStepWithWorkspaceFromState(),
		},
	})
}

func testAccCheckCustomRoleExists(name string) resource.TestCheckFunc {
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
		resp, err := cli.OrgCli.GetCustomRole(ctx, &orgcv1.GetCustomRoleRequest{
			OrganizationId: cli.OrgId,
			Id:             rs.Primary.ID,
		})
		if err != nil {
			return fmt.Errorf("GetCustomRole(%s) failed: %w", rs.Primary.ID, err)
		}
		if resp.GetCustomRole() == nil || resp.GetCustomRole().GetId() != rs.Primary.ID {
			return fmt.Errorf("GetCustomRole(%s) returned mismatched payload", rs.Primary.ID)
		}
		return nil
	}
}

func testAccCheckCustomRoleExistsWorkspace(name, workspaceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		cli := getTestAkpCli()
		if cli == nil {
			return fmt.Errorf("could not get test client")
		}
		ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
		ws, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, workspaceName)
		if err != nil {
			return fmt.Errorf("resolve workspace %q: %w", workspaceName, err)
		}
		resp, err := cli.OrgCli.GetWorkspaceCustomRole(ctx, &orgcv1.GetWorkspaceCustomRoleRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    ws.GetId(),
			Id:             rs.Primary.ID,
		})
		if err != nil {
			return fmt.Errorf("GetWorkspaceCustomRole(%s) failed: %w", rs.Primary.ID, err)
		}
		if resp.GetCustomRole() == nil || resp.GetCustomRole().GetId() != rs.Primary.ID {
			return fmt.Errorf("GetWorkspaceCustomRole(%s) returned mismatched payload", rs.Primary.ID)
		}
		return nil
	}
}

// testAccCheckCustomRoleDestroyed walks any leftover state and confirms each
// id resolves to NotFound. Covers both scoping flavors via the workspace
// attribute pulled off state.
func testAccCheckCustomRoleDestroyed(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "akp_custom_role" {
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

		workspaceName := rs.Primary.Attributes["workspace"]
		if workspaceName != "" {
			ws, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, workspaceName)
			if err != nil {
				if isGoneErr(err) {
					// Workspace gone too — nothing dangling.
					continue
				}
				return fmt.Errorf("resolve workspace %q during destroy check: %w", workspaceName, err)
			}
			_, err = cli.OrgCli.GetWorkspaceCustomRole(ctx, &orgcv1.GetWorkspaceCustomRoleRequest{
				OrganizationId: cli.OrgId,
				WorkspaceId:    ws.GetId(),
				Id:             rs.Primary.ID,
			})
			if err == nil {
				return fmt.Errorf("workspace custom role %s still exists after destroy", rs.Primary.ID)
			}
			if !isGoneErr(err) {
				return fmt.Errorf("GetWorkspaceCustomRole after destroy returned unexpected error: %w", err)
			}
			continue
		}

		_, err := cli.OrgCli.GetCustomRole(ctx, &orgcv1.GetCustomRoleRequest{
			OrganizationId: cli.OrgId,
			Id:             rs.Primary.ID,
		})
		if err == nil {
			return fmt.Errorf("custom role %s still exists after destroy", rs.Primary.ID)
		}
		if !isGoneErr(err) {
			return fmt.Errorf("GetCustomRole after destroy returned unexpected error: %w", err)
		}
	}
	return nil
}

func captureCustomRoleID(name string, id *string) resource.TestCheckFunc {
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

func expectCustomRoleSameID(name string, prevID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		if rs.Primary.ID != *prevID {
			return fmt.Errorf("expected id to remain %s across in-place update; got %s", *prevID, rs.Primary.ID)
		}
		return nil
	}
}

func testAccCustomRoleImportStateStep() resource.TestStep {
	return resource.TestStep{
		ResourceName:      customRoleResourceName,
		ImportState:       true,
		ImportStateVerify: true,
	}
}

// testAccCustomRoleImportStateStepWithWorkspaceFromState mirrors the api_key
// helper: derive the workspace name straight from state so the test doesn't
// need to know how the workspace was named at config-build time.
func testAccCustomRoleImportStateStepWithWorkspaceFromState() resource.TestStep {
	return resource.TestStep{
		ResourceName: customRoleResourceName,
		ImportState:  true,
		ImportStateIdFunc: func(s *terraform.State) (string, error) {
			rs, ok := s.RootModule().Resources[customRoleResourceName]
			if !ok {
				return "", fmt.Errorf("not found: %s", customRoleResourceName)
			}
			workspace := rs.Primary.Attributes["workspace"]
			if workspace == "" {
				return "", fmt.Errorf("workspace attribute missing on %s", customRoleResourceName)
			}
			return fmt.Sprintf("%s/%s", workspace, rs.Primary.ID), nil
		},
		ImportStateVerify: true,
	}
}

func testAccCustomRoleConfigOrg(name, description, policy string) string {
	return fmt.Sprintf(`
resource "akp_custom_role" "test" {
  name        = %q
  description = %q
  policy      = %q
}
`, name, description, policy)
}

func testAccCustomRoleConfigWorkspaceInline(workspaceName, name, description, policy string) string {
	return fmt.Sprintf(`
resource "akp_workspace" "test" {
  name = %q
}

resource "akp_custom_role" "test" {
  workspace   = akp_workspace.test.name
  name        = %q
  description = %q
  policy      = %q
}
`, workspaceName, name, description, policy)
}

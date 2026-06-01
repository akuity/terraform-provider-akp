//go:build !unit

package akp

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	apikeyv1 "github.com/akuity/api-client-go/pkg/api/gen/apikey/v1"
)

const apiKeyResourceName = "akp_api_key.test"

// runApiKeyResource creates, reads, imports, and replaces an org-scoped key.
// Replacement is exercised by changing description (RequiresReplace) — that
// must mint a new id and a new secret.
func runApiKeyResource(t *testing.T) {
	desc := fmt.Sprintf("tf-acc-%s", acctest.RandString(8))
	descUpdated := fmt.Sprintf("tf-acc-%s-upd", acctest.RandString(8))

	var firstID, firstSecret string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckApiKeyDestroyed(apiKeyResourceName),
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccApiKeyConfigOrg(desc, "member", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(apiKeyResourceName, "id"),
					resource.TestCheckResourceAttrSet(apiKeyResourceName, "secret"),
					resource.TestCheckResourceAttrSet(apiKeyResourceName, "create_time"),
					resource.TestCheckResourceAttr(apiKeyResourceName, "description", desc),
					resource.TestCheckResourceAttr(apiKeyResourceName, "permissions.roles.#", "1"),
					resource.TestCheckResourceAttr(apiKeyResourceName, "permissions.roles.0", "member"),
					resource.TestCheckResourceAttr(apiKeyResourceName, "expire_time", ""),
					testAccCheckApiKeyExists(apiKeyResourceName),
					captureApiKeyState(apiKeyResourceName, &firstID, &firstSecret),
				),
			},
			testAccApiKeyImportStateStep(),
			{
				// RequiresReplace on description should destroy + create. Verify
				// the id and secret both rotated so callers know a fresh key was
				// issued on update.
				Config: providerConfig + testAccApiKeyConfigOrg(descUpdated, "member", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(apiKeyResourceName, "description", descUpdated),
					expectApiKeyReplaced(apiKeyResourceName, &firstID, &firstSecret),
					testAccCheckApiKeyExists(apiKeyResourceName),
				),
			},
		},
	})
}

// runApiKeyResourceExpiring exercises the expire_in_duration path; the API
// translates the duration to an absolute expire_time we can verify.
func runApiKeyResourceExpiring(t *testing.T) {
	desc := fmt.Sprintf("tf-acc-exp-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckApiKeyDestroyed(apiKeyResourceName),
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccApiKeyConfigOrg(desc, "member", "1h"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(apiKeyResourceName, "id"),
					resource.TestCheckResourceAttrSet(apiKeyResourceName, "secret"),
					resource.TestCheckResourceAttr(apiKeyResourceName, "expire_in_duration", "1h"),
					resource.TestCheckResourceAttrSet(apiKeyResourceName, "expire_time"),
					testAccCheckApiKeyExists(apiKeyResourceName),
				),
			},
		},
	})
}

// runApiKeyResourceMissingRoles verifies that a permissions block without
// roles or custom_roles is rejected before any API call.
func runApiKeyResourceMissingRoles(t *testing.T) {
	desc := fmt.Sprintf("tf-acc-bad-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + testAccApiKeyConfigNoRoles(desc),
				ExpectError: regexp.MustCompile("permissions must include at least one entry in `roles` or `custom_roles`"),
			},
		},
	})
}

// runApiKeyResourceWorkspace covers the workspace-scoped path end-to-end by
// provisioning its own workspace inline. The api_key takes a hard dependency
// on the workspace via `workspace = akp_workspace.test.name`, so teardown
// happens in the right order automatically.
func runApiKeyResourceWorkspace(t *testing.T) {
	suffix := acctest.RandString(8)
	workspaceName := fmt.Sprintf("tf-acc-ws-%s", suffix)
	desc := fmt.Sprintf("tf-acc-ws-key-%s", suffix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckApiKeyDestroyed(apiKeyResourceName),
			testAccCheckWorkspaceDestroyed,
		),
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccApiKeyConfigWorkspaceInline(workspaceName, desc),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(apiKeyResourceName, "id"),
					resource.TestCheckResourceAttrSet(apiKeyResourceName, "secret"),
					resource.TestCheckResourceAttr(apiKeyResourceName, "workspace", workspaceName),
					resource.TestCheckResourceAttr(apiKeyResourceName, "description", desc),
					testAccCheckApiKeyExists(apiKeyResourceName),
				),
			},
			testAccApiKeyImportStateStepWithWorkspaceFromState(),
		},
	})
}

// testAccCheckApiKeyExists hits the apikey service directly and verifies the
// id from terraform state still resolves. Routes to the workspace endpoint
// when the resource is workspace-scoped — the top-level GetAPIKey enforces
// `organization/apikeys`, which is the wrong namespace for a workspace key.
func testAccCheckApiKeyExists(name string) resource.TestCheckFunc {
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

		workspaceName := rs.Primary.Attributes["workspace"]
		if workspaceName != "" {
			ws, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, workspaceName)
			if err != nil {
				return fmt.Errorf("resolve workspace %q for existence check: %w", workspaceName, err)
			}
			resp, err := cli.ApiKeyCli.GetWorkspaceAPIKey(ctx, &apikeyv1.GetWorkspaceAPIKeyRequest{
				OrganizationId: cli.OrgId,
				WorkspaceId:    ws.GetId(),
				Id:             rs.Primary.ID,
			})
			if err != nil {
				return fmt.Errorf("GetWorkspaceAPIKey(%s/%s) failed: %w", ws.GetId(), rs.Primary.ID, err)
			}
			if resp.GetApiKey() == nil || resp.GetApiKey().GetId() != rs.Primary.ID {
				return fmt.Errorf("GetWorkspaceAPIKey(%s) returned mismatched payload", rs.Primary.ID)
			}
			return nil
		}

		resp, err := cli.ApiKeyCli.GetAPIKey(ctx, &apikeyv1.GetAPIKeyRequest{Id: rs.Primary.ID})
		if err != nil {
			return fmt.Errorf("GetAPIKey(%s) failed: %w", rs.Primary.ID, err)
		}
		if resp.GetApiKey() == nil || resp.GetApiKey().GetId() != rs.Primary.ID {
			return fmt.Errorf("GetAPIKey(%s) returned mismatched payload", rs.Primary.ID)
		}
		return nil
	}
}

// testAccCheckApiKeyDestroyed verifies the key has been deleted server-side
// after the test step that removed it. Routes to the workspace endpoint when
// the resource is workspace-scoped — the top-level GetAPIKey enforces
// `organization/apikeys`, which always returns PermissionDenied for workspace
// keys (regardless of whether they still exist), so using it here can
// false-pass via isGoneErr.
func testAccCheckApiKeyDestroyed(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "akp_api_key" {
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
					// Workspace gone too — cascaded delete, nothing dangling.
					if isGoneErr(err) {
						continue
					}
					return fmt.Errorf("resolve workspace %q during destroy check: %w", workspaceName, err)
				}
				_, err = cli.ApiKeyCli.GetWorkspaceAPIKey(ctx, &apikeyv1.GetWorkspaceAPIKeyRequest{
					OrganizationId: cli.OrgId,
					WorkspaceId:    ws.GetId(),
					Id:             rs.Primary.ID,
				})
				if err == nil {
					return fmt.Errorf("workspace API key %s still exists after destroy", rs.Primary.ID)
				}
				if !isGoneErr(err) {
					return fmt.Errorf("GetWorkspaceAPIKey after destroy returned unexpected error: %w", err)
				}
				continue
			}

			_, err := cli.ApiKeyCli.GetAPIKey(ctx, &apikeyv1.GetAPIKeyRequest{Id: rs.Primary.ID})
			if err == nil {
				return fmt.Errorf("API key %s still exists after destroy", rs.Primary.ID)
			}
			if !isGoneErr(err) {
				return fmt.Errorf("GetAPIKey after destroy returned unexpected error: %w", err)
			}
		}
		return nil
	}
}

func captureApiKeyState(name string, id, secret *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		*id = rs.Primary.ID
		*secret = rs.Primary.Attributes["secret"]
		if *id == "" || *secret == "" {
			return fmt.Errorf("expected id and secret to be set; got id=%q secret-empty=%t", *id, *secret == "")
		}
		return nil
	}
}

func expectApiKeyReplaced(name string, prevID, prevSecret *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		if rs.Primary.ID == *prevID {
			return fmt.Errorf("expected id to change after replacement; still %s", rs.Primary.ID)
		}
		if rs.Primary.Attributes["secret"] == *prevSecret {
			return fmt.Errorf("expected secret to change after replacement")
		}
		return nil
	}
}

func testAccApiKeyImportStateStep() resource.TestStep {
	return resource.TestStep{
		ResourceName:      apiKeyResourceName,
		ImportState:       true,
		ImportStateVerify: true,
		// Server cannot echo the secret on read; expire_in_duration is config-only.
		ImportStateVerifyIgnore: []string{"secret", "expire_in_duration"},
	}
}

// testAccApiKeyImportStateStepWithWorkspaceFromState pulls the workspace name
// straight out of the resource's own state attribute, so it works with any
// inline-provisioned workspace without the test needing to hardcode the name.
func testAccApiKeyImportStateStepWithWorkspaceFromState() resource.TestStep {
	return resource.TestStep{
		ResourceName: apiKeyResourceName,
		ImportState:  true,
		ImportStateIdFunc: func(s *terraform.State) (string, error) {
			rs, ok := s.RootModule().Resources[apiKeyResourceName]
			if !ok {
				return "", fmt.Errorf("not found: %s", apiKeyResourceName)
			}
			workspace := rs.Primary.Attributes["workspace"]
			if workspace == "" {
				return "", fmt.Errorf("workspace attribute missing on %s", apiKeyResourceName)
			}
			return fmt.Sprintf("%s/%s", workspace, rs.Primary.ID), nil
		},
		ImportStateVerify:       true,
		ImportStateVerifyIgnore: []string{"secret", "expire_in_duration"},
	}
}

func testAccApiKeyConfigOrg(description, role, expireIn string) string {
	expireLine := ""
	if expireIn != "" {
		expireLine = fmt.Sprintf("  expire_in_duration = %q\n", expireIn)
	}
	return fmt.Sprintf(`
resource "akp_api_key" "test" {
  description = %q
%s  permissions = {
    roles = [%q]
  }
}
`, description, expireLine, role)
}

func testAccApiKeyConfigNoRoles(description string) string {
	return fmt.Sprintf(`
resource "akp_api_key" "test" {
  description = %q
  permissions = {
    actions = []
  }
}
`, description)
}

func testAccApiKeyConfigWorkspaceInline(workspaceName, description string) string {
	return fmt.Sprintf(`
resource "akp_workspace" "test" {
  name = %q
}

resource "akp_api_key" "test" {
  workspace   = akp_workspace.test.name
  description = %q
  permissions = {
    roles = ["member"]
  }
}
`, workspaceName, description)
}

//go:build !unit

package akp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/go-jose/go-jose/v4"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	serviceaccountv1 "github.com/akuity/api-client-go/pkg/api/gen/serviceaccount/v1"
)

const (
	oidcIssuerResourceName     = "akp_oidc_issuer.test"
	serviceAccountResourceName = "akp_service_account.test"
)

// skipUnlessOIDCFederation skips when the acceptance organization does not
// have the oidc_federation gate: every issuer and service account call is
// refused then, which is the gate working, not the resources failing.
func skipUnlessOIDCFederation(t *testing.T) {
	t.Helper()
	cli := getTestAkpCli()
	if cli == nil {
		t.Skip("no acceptance client")
	}
	ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
	_, err := cli.ServiceAccountCli.ListOIDCIssuers(ctx, &serviceaccountv1.ListOIDCIssuersRequest{OrganizationId: cli.OrgId})
	if err != nil && strings.Contains(err.Error(), "feature is not enabled") {
		t.Skip("oidc federation is not enabled for the acceptance organization")
	}
}

// testStaticJWKS is a fresh RSA signing key as a JWKS document, so the issuer
// needs no network to be probed.
func testStaticJWKS(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "acc-1", Algorithm: "RS256", Use: "sig"}}})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func runOIDCIssuerResource(t *testing.T) {
	skipUnlessOIDCFederation(t)
	name := fmt.Sprintf("tf-acc-%s", acctest.RandString(8))
	jwks := testStaticJWKS(t)
	var firstID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckOIDCIssuerDestroyed,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccOIDCIssuerConfig(name, jwks, nil, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(oidcIssuerResourceName, "id"),
					resource.TestCheckResourceAttrSet(oidcIssuerResourceName, "default_audience"),
					resource.TestCheckResourceAttrSet(oidcIssuerResourceName, "create_time"),
					resource.TestCheckResourceAttr(oidcIssuerResourceName, "name", name),
					resource.TestCheckResourceAttr(oidcIssuerResourceName, "key_source", "static_jwks"),
					resource.TestCheckResourceAttr(oidcIssuerResourceName, "token_ttl_seconds", "3600"),
					resource.TestCheckNoResourceAttr(oidcIssuerResourceName, "audiences.#"),
					testAccCheckOIDCIssuerExists(oidcIssuerResourceName),
					func(s *terraform.State) error {
						firstID = s.RootModule().Resources[oidcIssuerResourceName].Primary.ID
						return nil
					},
				),
			},
			{
				ResourceName:      oidcIssuerResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// Import has no config to keep, so static_jwks holds the
				// server's re-encoding of the same document.
				ImportStateVerifyIgnore: []string{"static_jwks"},
			},
			{
				// Audiences and TTL change in place; the ID must survive.
				Config: providerConfig + testAccOIDCIssuerConfig(name, jwks, []string{"api://acc"}, 600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(oidcIssuerResourceName, "audiences.#", "1"),
					resource.TestCheckResourceAttr(oidcIssuerResourceName, "audiences.0", "api://acc"),
					resource.TestCheckResourceAttr(oidcIssuerResourceName, "token_ttl_seconds", "600"),
					func(s *terraform.State) error {
						if got := s.RootModule().Resources[oidcIssuerResourceName].Primary.ID; got != firstID {
							return fmt.Errorf("expected an in-place update; id changed from %s to %s", firstID, got)
						}
						return nil
					},
					testAccCheckOIDCIssuerExists(oidcIssuerResourceName),
				),
			},
		},
	})
}

func runServiceAccountResource(t *testing.T) {
	skipUnlessOIDCFederation(t)
	suffix := acctest.RandString(8)
	jwks := testStaticJWKS(t)
	var firstID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckServiceAccountDestroyed,
			testAccCheckOIDCIssuerDestroyed,
		),
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccOIDCIssuerConfig("tf-acc-"+suffix, jwks, nil, 0) + testAccServiceAccountConfig("", "ci deployer", "member", "repo:acme/app:.*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(serviceAccountResourceName, "id"),
					resource.TestCheckResourceAttrSet(serviceAccountResourceName, "create_time"),
					resource.TestCheckResourceAttr(serviceAccountResourceName, "description", "ci deployer"),
					resource.TestCheckResourceAttr(serviceAccountResourceName, "disabled", "false"),
					resource.TestCheckResourceAttr(serviceAccountResourceName, "workspace_id", ""),
					resource.TestCheckResourceAttr(serviceAccountResourceName, "permissions.roles.#", "1"),
					resource.TestCheckResourceAttr(serviceAccountResourceName, "permissions.roles.0", "member"),
					resource.TestCheckResourceAttr(serviceAccountResourceName, "oidc_binding.match.sub", "repo:acme/app:.*"),
					resource.TestCheckResourceAttrPair(serviceAccountResourceName, "oidc_binding.issuer_id", oidcIssuerResourceName, "id"),
					testAccCheckServiceAccountExists(serviceAccountResourceName),
					func(s *terraform.State) error {
						firstID = s.RootModule().Resources[serviceAccountResourceName].Primary.ID
						return nil
					},
				),
			},
			{
				ResourceName:      serviceAccountResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Description and the binding change in place.
				Config: providerConfig + testAccOIDCIssuerConfig("tf-acc-"+suffix, jwks, nil, 0) + testAccServiceAccountConfig("", "ci deployer (main)", "member", "repo:acme/app:ref:refs/heads/main"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(serviceAccountResourceName, "description", "ci deployer (main)"),
					resource.TestCheckResourceAttr(serviceAccountResourceName, "oidc_binding.match.sub", "repo:acme/app:ref:refs/heads/main"),
					func(s *terraform.State) error {
						if got := s.RootModule().Resources[serviceAccountResourceName].Primary.ID; got != firstID {
							return fmt.Errorf("expected an in-place update; id changed from %s to %s", firstID, got)
						}
						return nil
					},
					testAccCheckServiceAccountExists(serviceAccountResourceName),
				),
			},
		},
	})
}

func runServiceAccountResourceWorkspace(t *testing.T) {
	skipUnlessOIDCFederation(t)
	suffix := acctest.RandString(8)
	workspaceName := "tf-acc-ws-" + suffix
	jwks := testStaticJWKS(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckServiceAccountDestroyed,
			testAccCheckOIDCIssuerDestroyed,
			testAccCheckWorkspaceDestroyed,
		),
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccOIDCIssuerConfig("tf-acc-"+suffix, jwks, nil, 0) +
					fmt.Sprintf("resource \"akp_workspace\" \"test\" {\n  name = %q\n}\n", workspaceName) +
					testAccServiceAccountConfig("akp_workspace.test.name", "", "admin", "repo:acme/app:.*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(serviceAccountResourceName, "id"),
					resource.TestCheckResourceAttr(serviceAccountResourceName, "workspace", workspaceName),
					resource.TestCheckResourceAttrPair(serviceAccountResourceName, "workspace_id", "akp_workspace.test", "id"),
					// The server adds organization/member; only the workspace role is state.
					resource.TestCheckResourceAttr(serviceAccountResourceName, "permissions.roles.#", "1"),
					resource.TestCheckResourceAttr(serviceAccountResourceName, "permissions.roles.0", "admin"),
					testAccCheckServiceAccountExists(serviceAccountResourceName),
				),
			},
			{
				ResourceName: serviceAccountResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[serviceAccountResourceName]
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["workspace"], rs.Primary.ID), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func testAccOIDCIssuerConfig(name, jwks string, audiences []string, ttl int) string {
	audienceLine := ""
	if audiences != nil {
		quoted := make([]string, len(audiences))
		for i, a := range audiences {
			quoted[i] = fmt.Sprintf("%q", a)
		}
		audienceLine = fmt.Sprintf("  audiences = [%s]\n", strings.Join(quoted, ", "))
	}
	ttlLine := ""
	if ttl > 0 {
		ttlLine = fmt.Sprintf("  token_ttl_seconds = %d\n", ttl)
	}
	return fmt.Sprintf(`
resource "akp_oidc_issuer" "test" {
  name        = %q
  issuer_url  = "https://issuer.acc.invalid"
  key_source  = "static_jwks"
  static_jwks = <<EOT
%s
EOT
%s%s}
`, name, jwks, audienceLine, ttlLine)
}

// testAccServiceAccountConfig binds the account to akp_oidc_issuer.test.
// workspaceRef is an HCL expression for the workspace name, or "" for an
// organization-level account.
func testAccServiceAccountConfig(workspaceRef, description, role, sub string) string {
	// An empty description is left out of the config: omitting it must work.
	lines := ""
	if workspaceRef != "" {
		lines += fmt.Sprintf("  workspace = %s\n", workspaceRef)
	}
	if description != "" {
		lines += fmt.Sprintf("  description = %q\n", description)
	}
	return fmt.Sprintf(`
resource "akp_service_account" "test" {
%s  permissions = {
    roles = [%q]
  }
  oidc_binding = {
    issuer_id = akp_oidc_issuer.test.id
    match = {
      sub = %q
    }
  }
}
`, lines, role, sub)
}

func testAccCheckOIDCIssuerExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok || rs.Primary.ID == "" {
			return fmt.Errorf("not found or no id: %s", name)
		}
		cli := getTestAkpCli()
		ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
		_, err := cli.ServiceAccountCli.GetOIDCIssuer(ctx, &serviceaccountv1.GetOIDCIssuerRequest{OrganizationId: cli.OrgId, Id: rs.Primary.ID})
		return err
	}
}

func testAccCheckOIDCIssuerDestroyed(s *terraform.State) error {
	cli := getTestAkpCli()
	ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "akp_oidc_issuer" || rs.Primary.ID == "" {
			continue
		}
		_, err := cli.ServiceAccountCli.GetOIDCIssuer(ctx, &serviceaccountv1.GetOIDCIssuerRequest{OrganizationId: cli.OrgId, Id: rs.Primary.ID})
		if err == nil {
			return fmt.Errorf("oidc issuer %s still exists", rs.Primary.ID)
		}
		if !isGoneErr(err) {
			return err
		}
	}
	return nil
}

// getServiceAccountForCheck routes to the scope's endpoint like the resource.
func getServiceAccountForCheck(ctx context.Context, cli *AkpCli, rs *terraform.ResourceState) error {
	if workspaceName := rs.Primary.Attributes["workspace"]; workspaceName != "" {
		ws, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, workspaceName)
		if err != nil {
			return err
		}
		_, err = cli.ServiceAccountCli.GetWorkspaceServiceAccount(ctx, &serviceaccountv1.GetWorkspaceServiceAccountRequest{OrganizationId: cli.OrgId, WorkspaceId: ws.GetId(), Id: rs.Primary.ID})
		return err
	}
	_, err := cli.ServiceAccountCli.GetServiceAccount(ctx, &serviceaccountv1.GetServiceAccountRequest{OrganizationId: cli.OrgId, Id: rs.Primary.ID})
	return err
}

func testAccCheckServiceAccountExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok || rs.Primary.ID == "" {
			return fmt.Errorf("not found or no id: %s", name)
		}
		cli := getTestAkpCli()
		ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
		return getServiceAccountForCheck(ctx, cli, rs)
	}
}

func testAccCheckServiceAccountDestroyed(s *terraform.State) error {
	cli := getTestAkpCli()
	ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "akp_service_account" || rs.Primary.ID == "" {
			continue
		}
		err := getServiceAccountForCheck(ctx, cli, rs)
		if err == nil {
			return fmt.Errorf("service account %s still exists", rs.Primary.ID)
		}
		if !isGoneErr(err) {
			return err
		}
	}
	return nil
}

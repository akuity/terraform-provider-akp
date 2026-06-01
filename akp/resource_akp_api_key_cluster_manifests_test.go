//go:build !unit

package akp

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
)

const clusterResourceName = "akp_cluster.test"

// runApiKeyResourceCustomRoleClusterManifests exercises the whole chain that
// an integrator builds in the UI: provision a cluster, mint a custom role
// whose Casbin policy grants GET only for that one cluster's manifests, bind
// the role to an API key, then prove the key can actually fetch the manifests
// via raw HTTP Basic auth — and nothing else.
func runApiKeyResourceCustomRoleClusterManifests(t *testing.T) {
	instanceID := getInstanceId()

	// The server resolves `workspace/instance/clusters` resources by looking
	// up the instance's actual workspace and rewriting the resource path
	// (organizationEnforcer.EnforceAction). Plain Casbin keyMatch can't span
	// that prefix with a wildcard, so we need the literal workspace id in the
	// policy.
	workspaceID, err := resolveInstanceWorkspaceID(instanceID)
	if err != nil {
		t.Fatalf("resolve workspace for instance %s: %v", instanceID, err)
	}

	suffix := acctest.RandString(8)
	clusterName := fmt.Sprintf("tf-acc-cm-%s", suffix)
	roleName := fmt.Sprintf("cluster-reader-%s", suffix)

	var apiKeyID, apiKeySecret, clusterID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckApiKeyDestroyed(apiKeyResourceName),
			testAccCheckCustomRoleDestroyed,
		),
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccApiKeyConfigCustomRoleClusterManifests(instanceID, workspaceID, clusterName, roleName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(clusterResourceName, "id"),
					resource.TestCheckResourceAttrSet(customRoleResourceName, "id"),
					resource.TestCheckResourceAttrSet(apiKeyResourceName, "id"),
					resource.TestCheckResourceAttrSet(apiKeyResourceName, "secret"),
					resource.TestCheckResourceAttr(apiKeyResourceName, "permissions.custom_roles.#", "1"),
					captureAttr(apiKeyResourceName, "id", &apiKeyID),
					captureAttr(apiKeyResourceName, "secret", &apiKeySecret),
					captureAttr(clusterResourceName, "id", &clusterID),
					testAccCheckClusterManifestsRetrievable(t, &apiKeyID, &apiKeySecret, instanceID, &clusterID),
				),
			},
		},
	})
}

// captureAttr is a generic state-grabbing helper. Test files keep redefining
// the same shape; this is the small reusable version.
func captureAttr(resourceName, attr string, out *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		v := rs.Primary.Attributes[attr]
		if v == "" {
			return fmt.Errorf("expected %s.%s to be set", resourceName, attr)
		}
		*out = v
		return nil
	}
}

// testAccCheckClusterManifestsRetrievable issues the same HTTP request the
// user would run from a shell — Basic auth with `api_key_id:secret`, GET the
// streaming manifests endpoint — and asserts a 200 with a non-empty body.
//
// The endpoint is a server-streaming RPC over grpc-gateway, so the body is a
// newline-delimited JSON envelope of base64-encoded HttpBody chunks. We don't
// decode it here; the auth check and "we got bytes" are what's load-bearing
// for proving the custom role's scope is correct.
func testAccCheckClusterManifestsRetrievable(t *testing.T, apiKeyID, apiKeySecret *string, instanceID string, clusterID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cli := getTestAkpCli()
		if cli == nil {
			return fmt.Errorf("could not get test client")
		}

		url := fmt.Sprintf("%s/api/v1/orgs/%s/argocd/instances/%s/clusters/%s/manifests",
			serverUrlForTests(), cli.OrgId, instanceID, *clusterID)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("build manifests request: %w", err)
		}
		creds := base64.StdEncoding.EncodeToString([]byte(*apiKeyID + ":" + *apiKeySecret))
		req.Header.Set("Authorization", "Basic "+creds)

		httpClient := &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				// Match the provider's AKUITY_SKIP_TLS_VERIFY behavior — tests
				// against local SaaS hit a self-signed cert.
				TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTLSVerify}, //nolint:gosec
			},
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("fetch cluster manifests: %w", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200 fetching manifests, got %d: %s", resp.StatusCode, string(body))
		}
		if len(body) == 0 {
			return fmt.Errorf("manifests body was empty; api key may lack effective grants for cluster %s", *clusterID)
		}
		t.Logf("cluster manifests retrieved via custom-role-bound API key: %d bytes", len(body))
		return nil
	}
}

func testAccApiKeyConfigCustomRoleClusterManifests(instanceID, workspaceID, clusterName, roleName string) string {
	// Casbin policy format is `p, sub, obj, act, resource` — 4 fields, no
	// trailing effect (the org model defaults to allow). Resource pattern is
	// `<workspaceID>/<instanceID>/<clusterName>` (FormatClusterResource →
	// "%s/%s/%s"). The server rewrites the workspace prefix at enforcement
	// time, so the policy MUST contain the actual workspace id — Casbin's
	// keyMatch can't span it with a wildcard.
	policy := fmt.Sprintf(
		"p, role:%s, workspace/instance/clusters, get, %s/%s/%s",
		roleName, workspaceID, instanceID, clusterName,
	)

	// HCL heredoc to sidestep quote-escaping inside the format() call. The
	// `${...}` references are HCL interpolation against resource attributes —
	// they survive Sprintf because `$`/`{` aren't format directives.
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name        = %q
  namespace   = "tf-acc-cm"
  spec = {
    namespace_scoped = true
    data = {
      size                  = "small"
      auto_upgrade_disabled = true
    }
  }
}

resource "akp_custom_role" "test" {
  name        = %q
  description = "Read manifests for the test cluster only"
  policy      = %q
}

resource "akp_api_key" "test" {
  description = "tf-acc cluster-manifests"
  permissions = {
    custom_roles = [akp_custom_role.test.id]
  }
}

// Demonstrates how an integrator turns the key into a usable HTTP call.
// Marked sensitive because the auth header carries the api-key secret.
output "manifests_curl_command" {
  sensitive = true
  value     = <<-EOT
    curl -H "Authorization: Basic ${base64encode("${akp_api_key.test.id}:${akp_api_key.test.secret}")}" '%s/api/v1/orgs/%s/argocd/instances/%s/clusters/${akp_cluster.test.id}/manifests'
  EOT
}
`, instanceID, clusterName, roleName, policy, serverUrlForTests(), getTestAkpCli().OrgId, instanceID)
}

// serverUrlForTests returns the AKUITY_SERVER_URL env value with the
// production default, mirroring getTestAkpCli's logic. Used both for the HTTP
// check and embedded in HCL output so the user can paste the rendered curl
// command directly into a shell.
func serverUrlForTests() string {
	if v := os.Getenv("AKUITY_SERVER_URL"); v != "" {
		return v
	}
	return "https://akuity.cloud"
}

// resolveInstanceWorkspaceID looks up the workspace the test instance lives
// in. The server uses this when enforcing `workspace/instance/clusters`
// permissions — see organizationEnforcer.EnforceAction.
func resolveInstanceWorkspaceID(instanceID string) (string, error) {
	cli := getTestAkpCli()
	if cli == nil {
		return "", fmt.Errorf("could not get test client")
	}
	ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
	resp, err := cli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
		OrganizationId: cli.OrgId,
		Id:             instanceID,
		IdType:         idv1.Type_ID,
	})
	if err != nil {
		return "", fmt.Errorf("GetInstance(%s): %w", instanceID, err)
	}
	ws := resp.GetInstance().GetWorkspaceId()
	if ws == "" {
		return "", fmt.Errorf("instance %s has empty workspace_id", instanceID)
	}
	return ws, nil
}

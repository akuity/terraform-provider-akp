//go:build !unit

package akp

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var (
	orgName        string
	providerConfig string
	skipTLSVerify  bool
)

func TestMain(m *testing.M) {
	if v := os.Getenv("AKUITY_SKIP_TLS_VERIFY"); v != "" {
		result, err := strconv.ParseBool(v)
		if err != nil {
			panic(err)
		}

		skipTLSVerify = result
	}

	if v := os.Getenv("AKUITY_ORG_NAME"); v == "" {
		orgName = "terraform-provider-acceptance-test"
	} else {
		orgName = v
	}

	providerConfig = fmt.Sprintf(`
provider "akp" {
	org_name = "%s"
	skip_tls_verify = %v
}
`, orgName, skipTLSVerify)

	code := m.Run()

	if os.Getenv("CLEANUP_TEST_INSTANCE") == "true" {
		cleanupTestInstance()
	}

	if os.Getenv("CLEANUP_TEST_KARGO_INSTANCE") == "true" {
		cleanupTestKargoInstance()
	}

	os.Exit(code)
}

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"akp": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("AKUITY_API_KEY_ID"); v == "" {
		t.Fatal("AKUITY_API_KEY_ID must be set for acceptance tests")
	}
	if v := os.Getenv("AKUITY_API_KEY_SECRET"); v == "" {
		t.Fatal("AKUITY_API_KEY_SECRET must be set for acceptance tests")
	}
}

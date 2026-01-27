//go:build !unit

package akp

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKargoInstanceResourceAdminAccountNonAlphabeticalValues(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargo-nonalpha-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with non-alphabetically ordered values
			// The provider should handle unordered values correctly using a set
			{
				Config: providerConfig + testAccKargoInstanceResourceConfigAdminAccountNonAlphabetical(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_instance.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.enabled", "true"),
					// Check that all values are present (order doesn't matter in a set)
					resource.TestCheckTypeSetElemAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.admin_account.claims.groups.values.*", "platform.infrastructure@foo.com"),
					resource.TestCheckTypeSetElemAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.admin_account.claims.groups.values.*", "oncall@foo.com"),
					resource.TestCheckTypeSetElemAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.admin_account.claims.groups.values.*", "sysadmin@foo.com"),
					resource.TestCheckTypeSetElemAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.admin_account.claims.groups.values.*", "security@foo.com"),
				),
			},
		},
	})
}

func testAccKargoInstanceResourceConfigAdminAccountNonAlphabetical(name string) string {
	return fmt.Sprintf(`
resource "akp_kargo_instance" "test" {
 name = %q
 kargo = {
   spec = {
     version = %q
     description = "Test Kargo instance with non-alphabetical admin account values"
     kargo_instance_spec = {
       backend_ip_allow_list_enabled = false
     }
     oidc_config = {
       enabled = true
       dex_enabled = false
       issuer_url = "https://test-issuer.example.com"
       client_id = "test-client-id"
       cli_client_id = "test-cli-client-id"

       admin_account = {
         claims = {
           groups = {
             values = [
               "platform.infrastructure@foo.com",
               "oncall@foo.com",
               "sysadmin@foo.com",
               "security@foo.com",
             ]
           }
         }
       }
     }
   }
 }
}
`, name, getKargoVersion())
}

func TestAccKargoInstanceResourceDexConfig(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargo-dex-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with dex_config
			{
				Config: providerConfig + testAccKargoInstanceResourceConfigDexConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_instance.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.enabled", "true"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.dex_enabled", "true"),
					resource.TestCheckResourceAttrSet("akp_kargo_instance.test", "kargo.spec.oidc_config.dex_config"),
				),
			},
		},
	})
}

func testAccKargoInstanceResourceConfigDexConfig(name string) string {
	return fmt.Sprintf(`
resource "akp_kargo_instance" "test" {
  name = %q
  kargo = {
    spec = {
      version = %q
      description = "Test Kargo instance with Dex configuration"
      kargo_instance_spec = {
        backend_ip_allow_list_enabled = false
      }
      oidc_config = {
        enabled = true
        dex_enabled = true

		dex_config_secret = {
    		"akp.dex.google.service.account" = "some-file"
    		"GOOGLE-CLIENT-SECRET"           = "some-secret"
  		}
        dex_config = yamlencode({
          connectors = [
            {
              id   = "google"
              type = "google"
              name = "Google"
              config = {
                clientID     = "some-id"
                clientSecret = "some-secret"
                adminEmail   = "argocd@foo.com"
                redirectURI  = "https://some-id.cd.akuity.cloud/api/dex/callback"
              }
            }
          ]
        })
      }
    }
  }
}
`, name, getKargoVersion())
}

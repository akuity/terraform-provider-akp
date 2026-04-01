//go:build !unit

package akp

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func runKargoConfigTests(t *testing.T) {
	name := getKargoInstanceName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Import the shared kargo instance
			{
				Config:            providerConfig + testAccKargoImportConfig(name),
				ImportState:       true,
				ImportStateId:     name,
				ResourceName:      "akp_kargo_instance.test",
				ImportStateVerify: false,
			},
			// Step 2: Admin Account Non-Alphabetical Values
			{
				Config: providerConfig + testAccKargoInstanceResourceConfigAdminAccountNonAlphabetical(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.enabled", "true"),
					resource.TestCheckTypeSetElemAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.admin_account.claims.groups.values.*", "platform.infrastructure@foo.com"),
					resource.TestCheckTypeSetElemAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.admin_account.claims.groups.values.*", "oncall@foo.com"),
				),
			},
			// Step 3: Dex Config
			{
				Config: providerConfig + testAccKargoInstanceResourceConfigDexConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.enabled", "true"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.dex_enabled", "true"),
					resource.TestCheckResourceAttrSet("akp_kargo_instance.test", "kargo.spec.oidc_config.dex_config"),
				),
			},
			testAccKargoImportStateStep(name, testAccKargoDexConfigSecretImportStateVerifyIgnore...),
			// Step 4: Spec and Config
			{
				Config: providerConfig + testAccKargoInstanceResourceConfigSpecAndConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.backend_ip_allow_list_enabled", "true"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.ip_allow_list.#", "2"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.promo_controller_enabled", "true"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.akuity_intelligence.enabled", "true"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.gc_config.max_retained_freight", "10"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.argocd_ui.idp_groups_mapping", "true"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test", "name", name),
					resource.TestCheckResourceAttrSet("data.akp_kargo_instance.test", "id"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.backend_ip_allow_list_enabled", "true"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo_resources.%", "1"),
					resource.TestCheckResourceAttrSet("akp_kargo_instance.test", "kargo_resources.kargo.akuity.io/v1alpha1/Project//test-project"),
				),
			},
			// Step 5: OIDC and Extras
			{
				Config: providerConfig + testAccKargoInstanceResourceConfigOIDCAndExtras(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.enabled", "true"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.additional_scopes.#", "2"),
					resource.TestCheckTypeSetElemAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.viewer_account.claims.groups.values.*", "viewer@example.com"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo_cm.adminAccountEnabled", "true"),
					resource.TestCheckResourceAttrSet("akp_kargo_instance.test", "kargo_secret.adminAccountPasswordHash"),
				),
			},
			testAccKargoImportStateStep(name, testAccKargoSecretImportStateVerifyIgnore...),
		},
	})
}

func runKargo_NestedOptionalObjectStability(t *testing.T) {
	name := acctest.RandomWithPrefix("kargo-nested-optional")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoInstanceResourceConfigNestedOptionalObjectStability(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.agent_customization_defaults.auto_upgrade_disabled", "true"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.akuity_intelligence.enabled", "true"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.gc_config.max_retained_freight", "10"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.enabled", "true"),
					resource.TestCheckTypeSetElemAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.viewer_account.claims.groups.values.*", "viewer@example.com"),
					resource.TestCheckNoResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.admin_account"),
					resource.TestCheckNoResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.user_account"),
					resource.TestCheckNoResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.project_creator_account"),
				),
			},
			{
				Config: providerConfig + testAccKargoInstanceResourceConfigNestedOptionalObjectStability(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.agent_customization_defaults.auto_upgrade_disabled", "true"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.akuity_intelligence.enabled", "true"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.gc_config.max_retained_freight", "10"),
					resource.TestCheckTypeSetElemAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.viewer_account.claims.groups.values.*", "viewer@example.com"),
					resource.TestCheckNoResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.admin_account"),
					resource.TestCheckNoResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.user_account"),
					resource.TestCheckNoResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.project_creator_account"),
				),
			},
			testAccKargoImportStateStep(name, testAccKargoInstanceCommonImportStateVerifyIgnore...),
		},
	})
}

func runKargo_MinimalSpecImport(t *testing.T) {
	name := acctest.RandomWithPrefix("kargo-minimal-spec")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoMinimalSpecConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_instance.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.promo_controller_enabled", "true"),
				),
			},
			{
				Config: providerConfig + testAccKargoMinimalSpecConfig(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			testAccKargoImportStateStep(name, testAccKargoMinimalImportStateVerifyIgnore...),
		},
	})
}

func testAccKargoMinimalSpecConfig(name string) string {
	return fmt.Sprintf(`
resource "akp_kargo_instance" "test" {
  name = %q
  kargo = {
    spec = {
      version     = %q
      description = "Minimal spec import test"
      kargo_instance_spec = {
        backend_ip_allow_list_enabled = true
        promo_controller_enabled      = true
      }
    }
  }
}`, name, getKargoVersion())
}

func runKargo_PartialOIDCImport(t *testing.T) {
	name := acctest.RandomWithPrefix("kargo-partial-oidc")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoPartialOIDCConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_instance.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.enabled", "true"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.issuer_url", "https://test-issuer.example.com"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.client_id", "test-client-id"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.cli_client_id", "test-cli-client-id"),
					resource.TestCheckNoResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.viewer_account"),
					resource.TestCheckNoResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.user_account"),
					resource.TestCheckNoResourceAttr("akp_kargo_instance.test", "kargo.spec.oidc_config.project_creator_account"),
				),
			},
			{
				Config: providerConfig + testAccKargoPartialOIDCConfig(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			testAccKargoImportStateStep(name, testAccKargoPartialOIDCImportStateVerifyIgnore...),
		},
	})
}

func testAccKargoPartialOIDCConfig(name string) string {
	return fmt.Sprintf(`
resource "akp_kargo_instance" "test" {
  name = %q
  kargo = {
    spec = {
      version = %q
      kargo_instance_spec = {
        backend_ip_allow_list_enabled = true
        promo_controller_enabled      = true
      }
      oidc_config = {
        enabled       = true
        dex_enabled   = false
        issuer_url    = "https://test-issuer.example.com"
        client_id     = "test-client-id"
        cli_client_id = "test-cli-client-id"
        admin_account = {
          claims = {
            groups = {
              values = ["admin@example.com"]
            }
          }
        }
      }
    }
  }
}`, name, getKargoVersion())
}

func runKargo_PartialKargoInstanceSpecImport(t *testing.T) {
	name := acctest.RandomWithPrefix("kargo-partial-spec")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoPartialKargoInstanceSpecConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_instance.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.akuity_intelligence.enabled", "true"),
					resource.TestCheckResourceAttr("akp_kargo_instance.test", "kargo.spec.kargo_instance_spec.promo_controller_enabled", "true"),
				),
			},
			{
				Config: providerConfig + testAccKargoPartialKargoInstanceSpecConfig(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			testAccKargoImportStateStep(name, testAccKargoPartialSpecImportStateVerifyIgnore...),
		},
	})
}

func testAccKargoPartialKargoInstanceSpecConfig(name string) string {
	return fmt.Sprintf(`
resource "akp_kargo_instance" "test" {
  name = %q
  kargo = {
    spec = {
      version     = %q
      description = "Partial kargo instance spec import test"
      kargo_instance_spec = {
        backend_ip_allow_list_enabled = true
        promo_controller_enabled      = true
        akuity_intelligence = {
          enabled = true
        }
        gc_config = {
          max_retained_freight = 5
        }
      }
    }
  }
}`, name, getKargoVersion())
}

// testAccKargoImportConfig returns a minimal config for importing the shared kargo instance.
func testAccKargoImportConfig(name string) string {
	return fmt.Sprintf(`
resource "akp_kargo_instance" "test" {
  name = %q
  kargo = {
    spec = {
      version = %q
    }
  }
}`, name, getKargoVersion())
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

func testAccKargoInstanceResourceConfigSpecAndConfig(name string) string {
	return fmt.Sprintf(`
resource "akp_kargo_instance" "test" {
  name = %q
  kargo = {
    spec = {
      version     = %q
      description = "Consolidated test: spec and config fields"
      kargo_instance_spec = {
        backend_ip_allow_list_enabled = true
        ip_allow_list = [
          {
            ip          = "192.168.1.0/24"
            description = "office network"
          },
          {
            ip          = "10.0.0.0/8"
            description = "internal network"
          },
        ]
        global_credentials_ns     = ["ns-creds-1", "ns-creds-2"]
        global_service_account_ns = ["ns-sa-1"]
        promo_controller_enabled  = true
        agent_customization_defaults = {
          auto_upgrade_disabled = true
          kustomization         = "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\n"
        }
        akuity_intelligence = {
          enabled                     = true
          ai_support_engineer_enabled = true
          allowed_usernames           = ["alice@example.com", "bob@example.com"]
          allowed_groups              = ["ai-users"]
        }
        gc_config = {
          max_retained_freight       = 10
          max_retained_promotions    = 20
          min_freight_deletion_age   = 3600
          min_promotion_deletion_age = 7200
        }
        argocd_ui = {
          idp_groups_mapping = true
        }
      }
    }
  }
  kargo_resources = {
    "kargo.akuity.io/v1alpha1/Project//test-project" = jsonencode({
      apiVersion = "kargo.akuity.io/v1alpha1"
      kind       = "Project"
      metadata = {
        name = "test-project"
      }
    })
  }
}

data "akp_kargo_instance" "test" {
  name = akp_kargo_instance.test.name
}`, name, getKargoVersion())
}

func testAccKargoInstanceResourceConfigOIDCAndExtras(name string) string {
	return fmt.Sprintf(`
resource "akp_kargo_instance" "test" {
  name = %q
  kargo = {
    spec = {
      version     = %q
      description = "Consolidated test: OIDC and extras"
      kargo_instance_spec = {
        backend_ip_allow_list_enabled = false
      }
      oidc_config = {
        enabled       = true
        dex_enabled   = false
        issuer_url    = "https://test-issuer.example.com"
        client_id     = "test-client-id"
        cli_client_id = "test-cli-client-id"
        additional_scopes = ["profile", "email"]
        viewer_account = {
          claims = {
            groups = {
              values = ["viewer@example.com"]
            }
          }
        }
        user_account = {
          claims = {
            groups = {
              values = ["user@example.com"]
            }
          }
        }
        project_creator_account = {
          claims = {
            groups = {
              values = ["creator@example.com"]
            }
          }
        }
      }
    }
  }
  kargo_cm = {
    adminAccountEnabled  = "true"
    adminAccountTokenTtl = "24h"
  }
  kargo_secret = {
    adminAccountPasswordHash = "$2a$10$wThs/VVwx5Tbygkk5Rzbv.V8hR8JYYmRdBiGjue9pd0YcEXl7.Kn."
  }
}`, name, getKargoVersion())
}

func testAccKargoInstanceResourceConfigNestedOptionalObjectStability(name string) string {
	return fmt.Sprintf(`
resource "akp_kargo_instance" "test" {
  name = %q
  kargo = {
    spec = {
      version     = %q
      description = "Nested optional object stability test"
      kargo_instance_spec = {
        backend_ip_allow_list_enabled = false
        agent_customization_defaults = {
          auto_upgrade_disabled = true
          kustomization         = "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\n"
        }
        akuity_intelligence = {
          enabled          = true
          allowed_groups   = ["platform"]
          allowed_usernames = ["viewer@example.com"]
        }
        gc_config = {
          max_retained_freight = 10
        }
      }
      oidc_config = {
        enabled       = true
        dex_enabled   = false
        issuer_url    = "https://test-issuer.example.com"
        client_id     = "test-client-id"
        cli_client_id = "test-cli-client-id"
        viewer_account = {
          claims = {
            groups = {
              values = ["viewer@example.com"]
            }
          }
        }
      }
    }
  }
}`, name, getKargoVersion())
}

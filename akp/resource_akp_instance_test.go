//go:build !unit

package akp

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// getTestInstanceVersion returns the ArgoCD version to use for acceptance tests.
// Uses AKUITY_ARGOCD_INSTANCE_VERSION env var if set, otherwise falls back to default.
func getTestInstanceVersion() string {
	if v := os.Getenv("AKUITY_ARGOCD_INSTANCE_VERSION"); v != "" {
		return v
	}
	return "v3.1.5-ak.65"
}

// TestAccInstanceResourceAIConfig tests the full AI configuration including
// kube_vision_config, ai_config, runbooks with slack_channel_names
func TestAccInstanceResourceAIConfig(t *testing.T) {
	name := fmt.Sprintf("test-ai-config-%s", acctest.RandString(8))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create instance with full AI configuration
			{
				Config: providerConfig + testAccInstanceResourceAIConfigFull(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_instance.test", "id"),
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),

					// Akuity Intelligence Extension
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.akuity_intelligence_extension.enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.akuity_intelligence_extension.allowed_usernames.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.akuity_intelligence_extension.allowed_usernames.0", "admin"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.akuity_intelligence_extension.allowed_usernames.1", "test-user"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.akuity_intelligence_extension.allowed_groups.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.akuity_intelligence_extension.allowed_groups.0", "admins"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.akuity_intelligence_extension.allowed_groups.1", "developers"),

					// KubeVision Config - CVE Scan
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.cve_scan_config.scan_enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.cve_scan_config.rescan_interval", "12h"),

					// KubeVision Config - AI Config
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_service", "slack-service"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_channels.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_channels.0", "alerts-channel"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_channels.1", "incidents-channel"),

					// Runbooks
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.#", "2"),

					// First runbook
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.name", "pod-crashloop-runbook"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.content", "Steps to debug CrashLoopBackOff pods"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.argocd_applications.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.argocd_applications.0", "my-app"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.k8s_namespaces.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.k8s_namespaces.0", "production"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.clusters.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.clusters.0", "prod-cluster"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.degraded_for", "5m"),
					// Slack channel names for first runbook
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.0", "oncall-team"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.1", "platform-alerts"),

					// Second runbook
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.1.name", "oom-killed-runbook"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.1.content", "Steps to handle OOMKilled containers"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.1.slack_channel_names.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.1.slack_channel_names.0", "memory-alerts"),
				),
			},
			// Update AI configuration - modify slack channels and runbooks
			{
				Config: providerConfig + testAccInstanceResourceAIConfigUpdated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),

					// Updated runbook slack channel names
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.#", "3"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.0", "oncall-team"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.1", "platform-alerts"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.2", "sre-team"),

					// Updated argocd slack channels
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_channels.#", "3"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_channels.2", "critical-channel"),
				),
			},
		},
	})
}

// TestAccInstanceResourceAIConfigMinimal tests AI configuration with minimal runbook settings
func TestAccInstanceResourceAIConfigMinimal(t *testing.T) {
	name := fmt.Sprintf("test-ai-minimal-%s", acctest.RandString(8))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create instance with minimal AI configuration (runbook without slack_channel_names)
			{
				Config: providerConfig + testAccInstanceResourceAIConfigMinimal(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_instance.test", "id"),
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),

					// Runbook without slack_channel_names should still work
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.name", "basic-runbook"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.content", "Basic runbook content"),
				),
			},
			// Add slack_channel_names to existing runbook
			{
				Config: providerConfig + testAccInstanceResourceAIConfigMinimalWithSlack(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.0", "new-channel"),
				),
			},
		},
	})
}

// TestAccInstanceResourceIncidentsConfig tests incidents configuration with triggers, webhooks, and grouping
func TestAccInstanceResourceIncidentsConfig(t *testing.T) {
	name := fmt.Sprintf("test-incidents-%s", acctest.RandString(8))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccInstanceResourceIncidentsConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_instance.test", "id"),
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),

					// Incidents triggers
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.0.argocd_applications.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.0.argocd_applications.0", "app-one"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.0.argocd_applications.1", "app-two"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.0.k8s_namespaces.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.0.k8s_namespaces.0", "default"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.0.degraded_for", "10m"),

					// Incidents webhooks
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.0.name", "pagerduty-webhook"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.0.description_path", "$.description"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.0.cluster_path", "$.cluster"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.0.k8s_namespace_path", "$.namespace"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.0.argocd_application_name_path", "$.app"),
				),
			},
		},
	})
}

// TestAccInstanceDataSourceAIConfig tests that the data source correctly reads AI configuration
func TestAccInstanceDataSourceAIConfig(t *testing.T) {
	name := fmt.Sprintf("test-ds-ai-%s", acctest.RandString(8))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccInstanceResourceAIConfigForDataSource(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify resource creation
					resource.TestCheckResourceAttrSet("akp_instance.test", "id"),

					// Verify data source reads the AI config correctly
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.#", "1"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.name", "data-source-test-runbook"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.#", "2"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.0", "ds-channel-1"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.1", "ds-channel-2"),
				),
			},
		},
	})
}

func testAccInstanceResourceAIConfigFull(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "AI Config acceptance test instance"
      instance_spec = {
        declarative_management_enabled = true

        akuity_intelligence_extension = {
          enabled           = true
          allowed_usernames = ["admin", "test-user"]
          allowed_groups    = ["admins", "developers"]
        }

        kube_vision_config = {
          cve_scan_config = {
            scan_enabled    = true
            rescan_interval = "12h"
          }
          ai_config = {
            argocd_slack_service  = "slack-service"
            argocd_slack_channels = ["alerts-channel", "incidents-channel"]
            runbooks = [
              {
                name    = "pod-crashloop-runbook"
                content = "Steps to debug CrashLoopBackOff pods"
                applied_to = {
                  argocd_applications = ["my-app"]
                  k8s_namespaces      = ["production"]
                  clusters            = ["prod-cluster"]
                  degraded_for        = "5m"
                }
                slack_channel_names = ["oncall-team", "platform-alerts"]
              },
              {
                name                = "oom-killed-runbook"
                content             = "Steps to handle OOMKilled containers"
                slack_channel_names = ["memory-alerts"]
              }
            ]
          }
        }
      }
    }
  }
}
`, name, getTestInstanceVersion())
}

func testAccInstanceResourceAIConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "AI Config acceptance test instance - updated"
      instance_spec = {
        declarative_management_enabled = true

        akuity_intelligence_extension = {
          enabled           = true
          allowed_usernames = ["admin", "test-user"]
          allowed_groups    = ["admins", "developers"]
        }

        kube_vision_config = {
          cve_scan_config = {
            scan_enabled    = true
            rescan_interval = "12h"
          }
          ai_config = {
            argocd_slack_service  = "slack-service"
            argocd_slack_channels = ["alerts-channel", "incidents-channel", "critical-channel"]
            runbooks = [
              {
                name    = "pod-crashloop-runbook"
                content = "Steps to debug CrashLoopBackOff pods - updated"
                applied_to = {
                  argocd_applications = ["my-app"]
                  k8s_namespaces      = ["production"]
                  clusters            = ["prod-cluster"]
                  degraded_for        = "5m"
                }
                slack_channel_names = ["oncall-team", "platform-alerts", "sre-team"]
              },
              {
                name                = "oom-killed-runbook"
                content             = "Steps to handle OOMKilled containers"
                slack_channel_names = ["memory-alerts"]
              }
            ]
          }
        }
      }
    }
  }
}
`, name, getTestInstanceVersion())
}

func testAccInstanceResourceAIConfigMinimal(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "Minimal AI Config test"
      instance_spec = {
        declarative_management_enabled = true

        kube_vision_config = {
          ai_config = {
            runbooks = [
              {
                name    = "basic-runbook"
                content = "Basic runbook content"
              }
            ]
          }
        }
      }
    }
  }
}
`, name, getTestInstanceVersion())
}

func testAccInstanceResourceAIConfigMinimalWithSlack(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "Minimal AI Config test with Slack"
      instance_spec = {
        declarative_management_enabled = true

        kube_vision_config = {
          ai_config = {
            runbooks = [
              {
                name                = "basic-runbook"
                content             = "Basic runbook content"
                slack_channel_names = ["new-channel"]
              }
            ]
          }
        }
      }
    }
  }
}
`, name, getTestInstanceVersion())
}

func testAccInstanceResourceIncidentsConfig(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "Incidents Config test"
      instance_spec = {
        declarative_management_enabled = true

        kube_vision_config = {
          ai_config = {
            incidents = {
              triggers = [
                {
                  argocd_applications = ["app-one", "app-two"]
                  k8s_namespaces      = ["default"]
                  clusters            = ["main-cluster"]
                  degraded_for        = "10m"
                }
              ]
              webhooks = [
                {
                  name                        = "pagerduty-webhook"
                  description_path            = "$.description"
                  cluster_path                = "$.cluster"
                  k8s_namespace_path          = "$.namespace"
                  argocd_application_name_path = "$.app"
                }
              ]
            }
          }
        }
      }
    }
  }
}
`, name, getTestInstanceVersion())
}

func testAccInstanceResourceAIConfigForDataSource(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "Data source AI Config test"
      instance_spec = {
        declarative_management_enabled = true

        kube_vision_config = {
          ai_config = {
            runbooks = [
              {
                name                = "data-source-test-runbook"
                content             = "Runbook for testing data source reading"
                slack_channel_names = ["ds-channel-1", "ds-channel-2"]
              }
            ]
          }
        }
      }
    }
  }
}

data "akp_instance" "test" {
  name = akp_instance.test.name
}
`, name, getTestInstanceVersion())
}

// TestAccInstanceResourceMetricsIngressPasswordHash tests that the sensitive
// metrics_ingress_password_hash field is preserved correctly through apply/refresh cycles.
// This test verifies the fix for "Provider produced inconsistent result after apply"
// error that occurred because the password hash is a write-only field not returned from the API.
func TestAccInstanceResourceMetricsIngressPasswordHash(t *testing.T) {
	name := fmt.Sprintf("test-metrics-auth-%s", acctest.RandString(8))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create instance with metrics_ingress_password_hash
			{
				Config: providerConfig + testAccInstanceResourceMetricsIngressPasswordHash(name, "test-bcrypt-hash-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_instance.test", "id"),
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					// Verify the password hash is stored in state (sensitive, so we check it exists)
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_username", "metrics-user"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "test-bcrypt-hash-1"),
				),
			},
			// Update description but keep the same password hash - should not cause state inconsistency
			{
				Config: providerConfig + testAccInstanceResourceMetricsIngressPasswordHashUpdatedDescription(name, "test-bcrypt-hash-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.description", "Updated description"),
					// Password hash should still be preserved
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "test-bcrypt-hash-1"),
				),
			},
			// Update the password hash itself
			{
				Config: providerConfig + testAccInstanceResourceMetricsIngressPasswordHashUpdatedDescription(name, "test-bcrypt-hash-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "test-bcrypt-hash-2"),
				),
			},
		},
	})
}

// TestAccInstanceResourceAIConfigWithIncidentsAndRunbooks tests the complete AI configuration
// including ai_config with runbooks, incidents (triggers, webhooks, grouping), and verifies
// state consistency through apply/refresh cycles.
// This test verifies the fix for state inconsistency when nested configs are not returned from API.
func TestAccInstanceResourceAIConfigWithIncidentsAndRunbooks(t *testing.T) {
	name := fmt.Sprintf("test-ai-full-%s", acctest.RandString(8))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create instance with full AI configuration including incidents webhooks
			{
				Config: providerConfig + testAccInstanceResourceAIConfigWithIncidentsAndRunbooks(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_instance.test", "id"),
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),

					// Verify AI config slack settings
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_service", "argo-notifications"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_channels.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_channels.0", ""),

					// Verify runbooks
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.name", "oom"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.argocd_applications.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.argocd_applications.0", "guestbook-*"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.k8s_namespaces.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.k8s_namespaces.0", "*"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.clusters.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.clusters.0", "prod-cluster"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.clusters.1", "staging-cluster"),

					// Verify incidents triggers
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.0.argocd_applications.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.0.argocd_applications.0", "guestbook-prod-oom"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.0.degraded_for", "2m"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.1.k8s_namespaces.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.1.k8s_namespaces.0", "production"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.1.degraded_for", "10m"),

					// Verify incidents webhooks
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.0.name", "slack-alert"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.0.description_path", "{.body.alerts[0].annotations.description}"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.0.cluster_path", "{.query.clusterName}"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.0.k8s_namespace_path", "{.body.alerts[0].labels.namespace}"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.0.argocd_application_name_path", ""),

					// Verify metrics ingress
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_username", "metrics-user"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "bcrypt-hashed-password"),
				),
			},
			// Update without changes - verify no state inconsistency (re-apply same config)
			{
				Config: providerConfig + testAccInstanceResourceAIConfigWithIncidentsAndRunbooks(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// All values should remain the same
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "bcrypt-hashed-password"),
				),
			},
		},
	})
}

func testAccInstanceResourceMetricsIngressPasswordHash(name, passwordHash string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "Metrics ingress password hash test"
      instance_spec = {
        declarative_management_enabled = true
        metrics_ingress_username       = "metrics-user"
        metrics_ingress_password_hash  = %q
      }
    }
  }
}
`, name, getTestInstanceVersion(), passwordHash)
}

func testAccInstanceResourceMetricsIngressPasswordHashUpdatedDescription(name, passwordHash string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "Updated description"
      instance_spec = {
        declarative_management_enabled = true
        metrics_ingress_username       = "metrics-user"
        metrics_ingress_password_hash  = %q
      }
    }
  }
}
`, name, getTestInstanceVersion(), passwordHash)
}

func testAccInstanceResourceAIConfigWithIncidentsAndRunbooks(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "Full AI config with incidents and metrics auth"
      instance_spec = {
        declarative_management_enabled = true

        kube_vision_config = {
          ai_config = {
            argocd_slack_service  = "argo-notifications"
            argocd_slack_channels = [""]

            runbooks = [
              {
                name    = "oom"
                content = <<-EOF
                  ## Out of memory runbook
                  Steps to handle OOMKilled containers
                EOF
                applied_to = {
                  argocd_applications = ["guestbook-*"]
                  k8s_namespaces      = ["*"]
                  clusters            = ["prod-cluster", "staging-cluster"]
                }
              }
            ]

            incidents = {
              triggers = [
                {
                  argocd_applications = ["guestbook-prod-oom"]
                  degraded_for        = "2m"
                },
                {
                  k8s_namespaces = ["production"]
                  clusters       = ["prod-cluster"]
                  degraded_for   = "10m"
                }
              ]
              webhooks = [
                {
                  name                         = "slack-alert"
                  description_path             = "{.body.alerts[0].annotations.description}"
                  cluster_path                 = "{.query.clusterName}"
                  k8s_namespace_path           = "{.body.alerts[0].labels.namespace}"
                  argocd_application_name_path = ""
                }
              ]
            }
          }
        }

        metrics_ingress_username      = "metrics-user"
        metrics_ingress_password_hash = "bcrypt-hashed-password"
      }
    }
  }
}
`, name, getTestInstanceVersion())
}

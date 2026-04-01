//go:build !unit

package akp

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func runInstanceConfigTests(t *testing.T) {
	name := getInstanceName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Import the shared instance
			{
				PreConfig:         func() { fmt.Fprintln(os.Stderr, "\n==== STEP 1/16: Import the shared instance ====") },
				Config:            providerConfig + testAccInstanceImportConfig(name),
				ImportState:       true,
				ImportStateId:     name,
				ResourceName:      "akp_instance.test",
				ImportStateVerify: false,
			},
			// Step 2: AI Config Full
			{
				PreConfig: func() { fmt.Fprintln(os.Stderr, "\n==== STEP 2/16: AI Config Full ====") },
				Config:    providerConfig + testAccInstanceResourceAIConfigFull(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.akuity_intelligence_extension.enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_channels.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.#", "2"),
				),
			},
			// Step 3: AI Config Updated
			{
				PreConfig: func() { fmt.Fprintln(os.Stderr, "\n==== STEP 3/16: AI Config Updated ====") },
				Config:    providerConfig + testAccInstanceResourceAIConfigUpdated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.#", "3"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_channels.#", "3"),
				),
			},
			// Step 4: AI Config Minimal
			{
				PreConfig: func() { fmt.Fprintln(os.Stderr, "\n==== STEP 4/16: AI Config Minimal ====") },
				Config:    providerConfig + testAccInstanceResourceAIConfigMinimal(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.name", "basic-runbook"),
				),
			},
			// Step 5: AI Config Minimal With Slack
			{
				PreConfig: func() { fmt.Fprintln(os.Stderr, "\n==== STEP 5/16: AI Config Minimal With Slack ====") },
				Config:    providerConfig + testAccInstanceResourceAIConfigMinimalWithSlack(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.#", "1"),
				),
			},
			// Step 6: Incidents Config
			{
				PreConfig: func() { fmt.Fprintln(os.Stderr, "\n==== STEP 6/16: Incidents Config ====") },
				Config:    providerConfig + testAccInstanceResourceIncidentsConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.#", "1"),
				),
			},
			// Step 7: Metrics Ingress Password Hash
			{
				PreConfig: func() { fmt.Fprintln(os.Stderr, "\n==== STEP 7/16: Metrics Ingress Password Hash ====") },
				Config:    providerConfig + testAccInstanceResourceMetricsIngressPasswordHash(name, "test-bcrypt-hash-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_username", "metrics-user"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "test-bcrypt-hash-1"),
				),
			},
			// Step 8: Metrics Ingress Password Hash Updated Description
			{
				PreConfig: func() {
					fmt.Fprintln(os.Stderr, "\n==== STEP 8/16: Metrics Ingress Password Hash Updated Description ====")
				},
				Config: providerConfig + testAccInstanceResourceMetricsIngressPasswordHashUpdatedDescription(name, "test-bcrypt-hash-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.description", "Updated description"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "test-bcrypt-hash-1"),
				),
			},
			// Step 9: Update password hash
			{
				PreConfig: func() { fmt.Fprintln(os.Stderr, "\n==== STEP 9/16: Update password hash ====") },
				Config:    providerConfig + testAccInstanceResourceMetricsIngressPasswordHashUpdatedDescription(name, "test-bcrypt-hash-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "test-bcrypt-hash-2"),
				),
			},
			// Step 10: AI Config with Incidents and Runbooks
			{
				PreConfig: func() { fmt.Fprintln(os.Stderr, "\n==== STEP 10/16: AI Config with Incidents and Runbooks ====") },
				Config:    providerConfig + testAccInstanceResourceAIConfigWithIncidentsAndRunbooks(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_service", "argo-notifications"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "bcrypt-hashed-password"),
				),
			},
			testAccInstanceImportStateStep(name, testAccInstanceMetricsImportStateVerifyIgnore...),
			// Step 11: Re-apply same config (verify no state inconsistency)
			{
				PreConfig: func() { fmt.Fprintln(os.Stderr, "\n==== STEP 11/16: Re-apply same config ====") },
				Config:    providerConfig + testAccInstanceResourceAIConfigWithIncidentsAndRunbooks(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "bcrypt-hashed-password"),
				),
			},
			// Step 12: Core Fields
			{
				PreConfig: func() { fmt.Fprintln(os.Stderr, "\n==== STEP 12/16: Core Fields ====") },
				Config:    providerConfig + testAccInstanceResourceConfigCoreFields(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.declarative_management_enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.image_updater_enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_cm.exec.enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_cm.exec.shells", "bash,sh,powershell,cmd"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_cm.ga.trackingid", "UA-12345-1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_cm.help.chatUrl", "https://mycorp.slack.com/argo-cd"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_cm.help.chatText", "Chat now!"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_cm.statusbadge.url", "https://cd-status.apps.argoproj.io/"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_cm.users.session.duration", "24h"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_cm.application.instanceLabelKey", "mycompany.com/appname"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_cm.ui.bannercontent", "Hello there!"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_cm.ui.bannerurl", "https://argoproj.github.io"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_cm.accounts.alice", "apiKey,login"),
					resource.TestCheckResourceAttr("akp_instance.test", "repo_credential_secrets.%", "2"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "name", name),
					resource.TestCheckResourceAttrSet("data.akp_instance.test", "id"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.declarative_management_enabled", "true"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.image_updater_enabled", "true"),

					// argocd_secret
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_secret.%", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_secret.dex.github.clientSecret", "my-github-oidc-secret"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_secret.webhook.github.secret", "shhhh! it's a github secret"),
					// argocd_notifications_secret
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_notifications_secret.%", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_notifications_secret.email-username", "test@argoproj.io"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_notifications_secret.email-password", "password"),
					// application_set_secret
					resource.TestCheckResourceAttr("akp_instance.test", "application_set_secret.%", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "application_set_secret.my-appset-secret", "xyz456"),
					// argocd_notifications_cm
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_notifications_cm.%", "3"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_notifications_cm.trigger.on-sync-status-unknown"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_notifications_cm.template.my-custom-template"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_notifications_cm.defaultTriggers"),
					// argocd_image_updater_config
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_image_updater_config.%", "3"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_image_updater_config.git.email", "akuitybot@akuity.io"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_image_updater_config.git.user", "akuitybot"),
					// argocd_image_updater_ssh_config
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_image_updater_ssh_config.%", "1"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_image_updater_ssh_config.config"),
					// argocd_image_updater_secret
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_image_updater_secret.%", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_image_updater_secret.my-docker-credentials", "abcd1234"),
					// argocd_tls_certs_cm
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_tls_certs_cm.%", "1"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_tls_certs_cm.server.example.com"),
					// argocd_rbac_cm
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_rbac_cm.%", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_rbac_cm.policy.default", "role:readonly"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_rbac_cm.policy.csv"),
					// repo_template_credential_secrets
					resource.TestCheckResourceAttr("akp_instance.test", "repo_template_credential_secrets.%", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "repo_template_credential_secrets.repo-argoproj-https-creds.%", "4"),
					resource.TestCheckResourceAttr("akp_instance.test", "repo_template_credential_secrets.repo-argoproj-https-creds.url", "https://github.com/argoproj"),
					resource.TestCheckResourceAttr("akp_instance.test", "repo_template_credential_secrets.repo-argoproj-https-creds.type", "helm"),
					resource.TestCheckResourceAttr("akp_instance.test", "repo_template_credential_secrets.repo-argoproj-https-creds.username", "my-username"),
					resource.TestCheckResourceAttr("akp_instance.test", "repo_template_credential_secrets.repo-argoproj-https-creds.password", "my-password"),
					// argocd_resources
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_resources.%", "1"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_resources.argoproj.io/v1alpha1/AppProject//test-project"),
					// argocd_ssh_known_hosts_cm
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_ssh_known_hosts_cm.%", "1"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_ssh_known_hosts_cm.ssh_known_hosts"),
				),
			},
			testAccInstanceImportStateStep(name, testAccInstanceCoreFieldsImportStateVerifyIgnore...),
			// Step 13: Spec Features
			{
				PreConfig: func() {
					fmt.Fprintln(os.Stderr, "\n==== STEP 13/16: Spec Features ====")
					time.Sleep(30 * time.Second)
				},
				Config: providerConfig + testAccInstanceResourceConfigSpecFeatures(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.extensions.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.cluster_customization_defaults.auto_upgrade_disabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.appset_policy.policy", "create-update"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.manifest_generation.kustomize.default_version", "v5.4.3"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.manifest_generation.kustomize.additional_versions.#", "2"),
				),
			},
			// Step 14: Misc Features
			{
				PreConfig: func() { fmt.Fprintln(os.Stderr, "\n==== STEP 14/16: Misc Features ====") },
				Config:    providerConfig + testAccInstanceResourceConfigMiscFeatures(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.host_aliases.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.crossplane_extension.resources.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "config_management_plugins.test-plugin.enabled", "true"),
				),
			},
			// Step 15: CMP Create
			{
				PreConfig: func() {
					fmt.Fprintln(os.Stderr, "\n==== STEP 15/16: CMP Create ====")
					time.Sleep(30 * time.Second)
				},
				Config: providerConfig + testAccInstanceResourceConfigCMP(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "config_management_plugins.my-plugin.enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "config_management_plugins.my-plugin.image", "busybox:latest"),
					resource.TestCheckResourceAttr("akp_instance.test", "config_management_plugins.my-plugin.spec.version", "v1.0"),
				),
			},
			// Step 16: CMP Update
			{
				PreConfig: func() { fmt.Fprintln(os.Stderr, "\n==== STEP 16/19: CMP Update ====") },
				Config:    providerConfig + testAccInstanceResourceConfigCMPUpdate(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "config_management_plugins.my-plugin.image", "alpine:latest"),
					resource.TestCheckResourceAttr("akp_instance.test", "config_management_plugins.my-plugin.spec.version", "v2.0"),
				),
			},
			// Step 17: IgnoreResourceUpdates
			{
				PreConfig: func() {
					fmt.Fprintln(os.Stderr, "\n==== STEP 17/19: IgnoreResourceUpdates config ====")
				},
				Config: providerConfig + testAccInstanceResourceIgnoreResourceUpdates(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_cm.resource.ignoreResourceUpdatesEnabled", "true"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_cm.resource.customizations.ignoreResourceUpdates.all"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_cm.resource.customizations.ignoreResourceUpdates.argoproj.io_Application"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_cm.resource.customizations.ignoreDifferences.admissionregistration.k8s.io_MutatingWebhookConfiguration"),
				),
			},
			// Step 18: Re-apply same config (verify no drift)
			{
				PreConfig: func() {
					fmt.Fprintln(os.Stderr, "\n==== STEP 18/19: Re-apply ignoreResourceUpdates (no drift) ====")
				},
				Config: providerConfig + testAccInstanceResourceIgnoreResourceUpdates(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			// Step 19: Remove customizations
			{
				PreConfig: func() {
					fmt.Fprintln(os.Stderr, "\n==== STEP 19/19: Remove customizations ====")
				},
				Config: providerConfig + testAccInstanceResourceNoCustomizations(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
				),
			},
		},
	})
}

func runInstance_NestedOptionalObjectStability(t *testing.T) {
	name := acctest.RandomWithPrefix("instance-nested-optional")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { fmt.Fprintln(os.Stderr, "\n==== INSTANCE NESTED OPTIONAL STABILITY 1/2: Create config ====") },
				Config:    providerConfig + testAccInstanceResourceNestedOptionalObjectStability(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_username", "metrics-user"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "bcrypt-hashed-password"),
					resource.TestCheckNoResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.grouping.%"),
					resource.TestCheckNoResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.%"),
				),
			},
			{
				PreConfig: func() {
					fmt.Fprintln(os.Stderr, "\n==== INSTANCE NESTED OPTIONAL STABILITY 2/2: Re-apply same config ====")
				},
				Config: providerConfig + testAccInstanceResourceNestedOptionalObjectStability(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "bcrypt-hashed-password"),
					resource.TestCheckNoResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.grouping.%"),
					resource.TestCheckNoResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.%"),
				),
			},
			testAccInstanceImportStateStep(name, testAccInstanceMetricsImportStateVerifyIgnore...),
		},
	})
}

func testAccInstanceResourceIgnoreResourceUpdates(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version = %q
      instance_spec = {
        declarative_management_enabled = true
        manifest_generation = {
          kustomize = {
            default_version = "v5.4.3"
          }
        }
      }
    }
  }
  argocd_cm = {
    "exec.enabled"  = "true"
    "helm.enabled"  = "true"
    "resource.ignoreResourceUpdatesEnabled" = "true"

    "resource.customizations.ignoreResourceUpdates.all" = <<-EOF
      jsonPointers:
        - /status
    EOF

    "resource.customizations.ignoreResourceUpdates.argoproj.io_Application" = <<-EOF
      jqPathExpressions:
        - '.metadata.annotations."notified.notifications.argoproj.io"'
        - '.metadata.annotations."argocd.argoproj.io/refresh"'
        - '.metadata.annotations."argocd.argoproj.io/hydrate"'
        - '.operation'
    EOF

    "resource.customizations.ignoreDifferences.admissionregistration.k8s.io_MutatingWebhookConfiguration" = <<-EOF
      jsonPointers:
        - /webhooks/0/clientConfig/caBundle
    EOF
  }
}
`, name, getInstanceVersion())
}

func testAccInstanceResourceNoCustomizations(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version = %q
      instance_spec = {
        declarative_management_enabled = true
        manifest_generation = {
          kustomize = {
            default_version = "v5.4.3"
          }
        }
      }
    }
  }
  argocd_cm = {
    "exec.enabled" = "true"
    "helm.enabled" = "true"
  }
}
`, name, getInstanceVersion())
}

// testAccInstanceImportConfig returns a minimal config for importing the shared instance.
func testAccInstanceImportConfig(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version = %q
    }
  }
}`, name, getInstanceVersion())
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
`, name, getInstanceVersion())
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
`, name, getInstanceVersion())
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
`, name, getInstanceVersion())
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
`, name, getInstanceVersion())
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
                  name                                = "pagerduty-webhook"
                  description_path                    = "$.description"
                  cluster_path                        = "$.cluster"
                  k8s_namespace_path                  = "$.namespace"
                  argocd_application_name_path        = "$.app"
                  argocd_application_namespace_path   = "$.appNamespace"
                }
              ]
            }
          }
        }
      }
    }
  }
}
`, name, getInstanceVersion())
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
`, name, getInstanceVersion(), passwordHash)
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
`, name, getInstanceVersion(), passwordHash)
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
                  name                              = "slack-alert"
                  description_path                  = "{.body.alerts[0].annotations.description}"
                  cluster_path                      = "{.query.clusterName}"
                  k8s_namespace_path                = "{.body.alerts[0].labels.namespace}"
                  argocd_application_name_path      = ""
                  argocd_application_namespace_path = "{.body.alerts[0].labels.app_namespace}"
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
`, name, getInstanceVersion())
}

func testAccInstanceResourceNestedOptionalObjectStability(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "Nested optional object stability test"
      instance_spec = {
        declarative_management_enabled = true

        kube_vision_config = {
          ai_config = {
            argocd_slack_service  = "argo-notifications"
            argocd_slack_channels = ["alerts"]

            runbooks = [
              {
                name    = "oom-killed-runbook"
                content = "Steps to handle OOMKilled containers"
              },
              {
                name    = "stuck-sync-runbook"
                content = "Steps to handle stuck syncs"
                applied_to = {
                  argocd_applications = ["guestbook-*"]
                  clusters            = ["prod-cluster"]
                }
              }
            ]

            incidents = {
              triggers = [
                {
                  k8s_namespaces = ["production"]
                  degraded_for   = "10m"
                }
              ]
              webhooks = [
                {
                  name                         = "slack-alert"
                  description_path             = "{.body.alerts[0].annotations.description}"
                  cluster_path                 = "{.query.clusterName}"
                  k8s_namespace_path           = "{.body.alerts[0].labels.namespace}"
                  argocd_application_namespace_path = "{.body.alerts[0].labels.app_namespace}"
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
`, name, getInstanceVersion())
}

func testAccInstanceResourceConfigCoreFields(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "Consolidated test: core fields"
      instance_spec = {
        declarative_management_enabled      = true
        image_updater_enabled               = true
        backend_ip_allow_list_enabled       = true
        audit_extension_enabled             = true
        sync_history_extension_enabled      = true
        multi_cluster_k8s_dashboard_enabled = true
        assistant_extension_enabled         = true
      }
    }
  }
  argocd_cm = {
    "exec.enabled"                   = true
    "exec.shells"                    = "bash,sh,powershell,cmd"
    "ga.trackingid"                  = "UA-12345-1"
    "ga.anonymizeusers"              = false
    "helm.enabled"                   = true
    "help.chatText"                  = "Chat now!"
    "help.chatUrl"                   = "https://mycorp.slack.com/argo-cd"
    "kustomize.enabled"              = true
    "server.rbac.log.enforce.enable" = false
    "statusbadge.enabled"            = false
    "statusbadge.url"                = "https://cd-status.apps.argoproj.io/"
    "application.instanceLabelKey"   = "mycompany.com/appname"
    "ui.bannercontent"               = "Hello there!"
    "ui.bannerpermanent"             = false
    "ui.bannerurl"                   = "https://argoproj.github.io"
    "users.anonymous.enabled"        = true
    "users.session.duration"         = "24h"

    "kustomize.buildOptions" = "--load_restrictor none"
    "accounts.alice"         = "apiKey,login"
    "dex.config" = <<-EOF
        connectors:
          # GitHub example
          - type: github
            id: github
            name: GitHub
            config:
              clientID: aabbccddeeff00112233
              clientSecret: $dex.github.clientSecret
              orgs:
              - name: your-github-org
        EOF
    "resource.customizations.ignoreDifferences.admissionregistration.k8s.io_MutatingWebhookConfiguration" = <<-EOF
        jsonPointers:
        - /webhooks/0/clientConfig/caBundle
        jqPathExpressions:
        - .webhooks[0].clientConfig.caBundle
        managedFieldsManagers:
        - kube-controller-manager
      EOF
    "resource.customizations.health.certmanager.k8s.io_Certificate" = <<-EOF
      hs = {}
      if obj.status ~= nil then
        if obj.status.conditions ~= nil then
          for i, condition in ipairs(obj.status.conditions) do
            if condition.type == "Ready" and condition.status == "False" then
              hs.status = "Degraded"
              hs.message = condition.message
              return hs
            end
            if condition.type == "Ready" and condition.status == "True" then
              hs.status = "Healthy"
              hs.message = condition.message
              return hs
            end
          end
        end
      end
      hs.status = "Progressing"
      hs.message = "Waiting for certificate"
      return hs
      EOF
    "resource.customizations.actions.apps_Deployment" = <<-EOF
      # Lua Script to indicate which custom actions are available on the resource
      discovery.lua: |
        actions = {}
        actions["restart"] = {}
        return actions
      definitions:
        - name: restart
          # Lua Script to modify the obj
          action.lua: |
            local os = require("os")
            if obj.spec.template.metadata == nil then
                obj.spec.template.metadata = {}
            end
            if obj.spec.template.metadata.annotations == nil then
                obj.spec.template.metadata.annotations = {}
            end
            obj.spec.template.metadata.annotations["kubectl.kubernetes.io/restartedAt"] = os.date("!%%Y-%%m-%%dT%%XZ")
            return obj
      EOF
    "resource.customizations.knownTypeFields.apps_StatefulSet" = <<-EOF
      - field: spec.volumeClaimTemplates
        type: array
      - field: spec.updateStrategy
        type: object
      EOF
  }
  argocd_rbac_cm = {
    "policy.default" = "role:readonly"
    "policy.csv" = <<-EOF
         p, role:org-admin, applications, *, */*, allow
         p, role:org-admin, clusters, get, *, allow
         g, your-github-org:your-team, role:org-admin
         EOF
  }
  argocd_secret = {
    "dex.github.clientSecret" = "my-github-oidc-secret"
    "webhook.github.secret"   = "shhhh! it's a github secret"
  }
  application_set_secret = {
    "my-appset-secret" = "xyz456"
  }
  argocd_notifications_cm = {
    "trigger.on-sync-status-unknown" = <<-EOF
        - when: app.status.sync.status == 'Unknown'
          send: [my-custom-template]
      EOF
    "template.my-custom-template" = <<-EOF
        message: |
          Application details: {{.context.argocdUrl}}/applications/{{.app.metadata.name}}.
      EOF
    "defaultTriggers" = <<-EOF
        - on-sync-status-unknown
      EOF
  }
  argocd_notifications_secret = {
    email-username = "test@argoproj.io"
    email-password = "password"
  }
  argocd_image_updater_ssh_config = {
    config = <<-EOF
      Host *
            PubkeyAcceptedAlgorithms +ssh-rsa
            HostkeyAlgorithms +ssh-rsa
            HostkeyAlgorithms2 +ssh-rsa
    EOF
  }
  argocd_image_updater_config = {
    "registries.conf" = <<-EOF
      registries:
        - prefix: docker.io
          name: Docker2
          api_url: https://registry-1.docker.io
          credentials: secret:argocd/argocd-image-updater-secret#my-docker-credentials
    EOF
    "git.email" = "akuitybot@akuity.io"
    "git.user"  = "akuitybot"
  }
  argocd_image_updater_secret = {
    my-docker-credentials = "abcd1234"
  }
  argocd_tls_certs_cm = {
    "server.example.com" = <<EOF
          -----BEGIN CERTIFICATE-----
          ......
          -----END CERTIFICATE-----
      EOF
  }
  repo_credential_secrets = {
    repo-my-private-https-repo = {
      url                = "https://github.com/argoproj/argocd-example-apps"
      password           = "my-password"
      username           = "my-username"
      insecure           = true
      forceHttpBasicAuth = true
      enableLfs          = true
    }
    repo-my-private-ssh-repo = {
      url           = "ssh://git@github.com/argoproj/argocd-example-apps"
      sshPrivateKey = <<EOF
      # paste the sshPrivateKey data here
      EOF
      insecure      = true
      enableLfs     = true
    }
  }
  repo_template_credential_secrets = {
    repo-argoproj-https-creds = {
      url      = "https://github.com/argoproj"
      type     = "helm"
      password = "my-password"
      username = "my-username"
    }
  }
  argocd_resources = {
    "argoproj.io/v1alpha1/AppProject//test-project" = jsonencode({
      apiVersion = "argoproj.io/v1alpha1"
      kind       = "AppProject"
      metadata = {
        name      = "test-project"
        namespace = "argocd"
      }
      spec = {
        description = "Test project"
        sourceRepos = ["*"]
        destinations = [{
          server    = "*"
          namespace = "*"
        }]
      }
    })
  }
  argocd_ssh_known_hosts_cm = {
    ssh_known_hosts = <<EOF
[ssh.github.com]:443 ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg=
[ssh.github.com]:443 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl
[ssh.github.com]:443 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQowgcQnjshcLrqPEiiphnt+VTTvDP6mHBL9j1aNUkY4Ue1gvwnGLVlOhGeYrnZaMgRK6+PKCUXaDbC7qtbW8gIkhL7aGCsOr/C56SJMy/BCZfxd1nWzAOxSDPgVsmerOBYfNqltV9/hWCqBywINIR+5dIg6JTJ72pcEpEjcYgXkE2YEFXV1JHnsKgbLWNlhScqb2UmyRkQyytRLtL+38TGxkxCflmO+5Z8CSSNY7GidjMIZ7Q4zMjA2n1nGrlTDkzwDCsw+wqFPGQA179cnfGWOWRVruj16z6XyvxvjJwbz0wQZ75XK5tKSb7FNyeIEs4TT4jk+S4dhPeAUC5y+bDYirYgM4GC7uEnztnZyaVWQ7B381AK4Qdrwt51ZqExKbQpTUNn+EjqoTwvqNj4kqx5QUCI0ThS/YkOxJCXmPUWZbhjpCg56i+2aB6CmK2JGhn57K5mj0MNdBXA4/WnwH6XoPWJzK5Nyu2zB3nAZp+S5hpQs+p1vN1/wsjk=
bitbucket.org ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBPIQmuzMBuKdWeF4+a2sjSSpBK0iqitSQ+5BM9KhpexuGt20JpTVM7u5BDZngncgrqDMbWdxMWWOGtZ9UgbqgZE=
bitbucket.org ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIazEu89wgQZ4bqs3d63QSMzYVa0MuJ2e2gKTKqu+UUO
bitbucket.org ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDQeJzhupRu0u0cdegZIa8e86EG2qOCsIsD1Xw0xSeiPDlCr7kq97NLmMbpKTX6Esc30NuoqEEHCuc7yWtwp8dI76EEEB1VqY9QJq6vk+aySyboD5QF61I/1WeTwu+deCbgKMGbUijeXhtfbxSxm6JwGrXrhBdofTsbKRUsrN1WoNgUa8uqN1Vx6WAJw1JHPhglEGGHea6QICwJOAr/6mrui/oB7pkaWKHj3z7d1IC4KWLtY47elvjbaTlkN04Kc/5LFEirorGYVbt15kAUlqGM65pk6ZBxtaO3+30LVlORZkxOh+LKL/BvbZ/iRNhItLqNyieoQj/uh/7Iv4uyH/cV/0b4WDSd3DptigWq84lJubb9t/DnZlrJazxyDCulTmKdOR7vs9gMTo+uoIrPSb8ScTtvw65+odKAlBj59dhnVp9zd7QUojOpXlL62Aw56U4oO+FALuevvMjiWeavKhJqlR7i5n9srYcrNV7ttmDw7kf/97P5zauIhxcjX+xHv4M=
github.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg=
github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl
github.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQowgcQnjshcLrqPEiiphnt+VTTvDP6mHBL9j1aNUkY4Ue1gvwnGLVlOhGeYrnZaMgRK6+PKCUXaDbC7qtbW8gIkhL7aGCsOr/C56SJMy/BCZfxd1nWzAOxSDPgVsmerOBYfNqltV9/hWCqBywINIR+5dIg6JTJ72pcEpEjcYgXkE2YEFXV1JHnsKgbLWNlhScqb2UmyRkQyytRLtL+38TGxkxCflmO+5Z8CSSNY7GidjMIZ7Q4zMjA2n1nGrlTDkzwDCsw+wqFPGQA179cnfGWOWRVruj16z6XyvxvjJwbz0wQZ75XK5tKSb7FNyeIEs4TT4jk+S4dhPeAUC5y+bDYirYgM4GC7uEnztnZyaVWQ7B381AK4Qdrwt51ZqExKbQpTUNn+EjqoTwvqNj4kqx5QUCI0ThS/YkOxJCXmPUWZbhjpCg56i+2aB6CmK2JGhn57K5mj0MNdBXA4/WnwH6XoPWJzK5Nyu2zB3nAZp+S5hpQs+p1vN1/wsjk=
gitlab.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBFSMqzJeV9rUzU4kWitGjeR4PWSa29SPqJ1fVkhtj3Hw9xjLVXVYrU9QlYWrOLXBpQ6KWjbjTDTdDkoohFzgbEY=
gitlab.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAfuCHKVTjquxvt6CM6tdG4SLp1Btn/nOeHHE5UOzRdf
gitlab.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCsj2bNKTBSpIYDEGk9KxsGh3mySTRgMtXL583qmBpzeQ+jqCMRgBqB98u3z++J1sKlXHWfM9dyhSevkMwSbhoR8XIq/U0tCNyokEi/ueaBMCvbcTHhO7FcwzY92WK4Yt0aGROY5qX2UKSeOvuP4D6TPqKF1onrSzH9bx9XUf2lEdWT/ia1NEKjunUqu1xOB/StKDHMoX4/OKyIzuS0q/T1zOATthvasJFoPrAjkohTyaDUz2LN5JoH839hViyEG82yB+MjcFV5MU3N1l1QL3cVUCh93xSaua1N85qivl+siMkPGbO5xR/En4iEY6K2XPASUEMaieWVNTRCtJ4S8H+9
ssh.dev.azure.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7Hr1oTWqNqOlzGJOfGJ4NakVyIzf1rXYd4d7wo6jBlkLvCA4odBlL0mDUyZ0/QUfTTqeu+tm22gOsv+VrVTMk6vwRU75gY/y9ut5Mb3bR5BV58dKXyq9A9UeB5Cakehn5Zgm6x1mKoVyf+FFn26iYqXJRgzIZZcZ5V6hrE0Qg39kZm4az48o0AUbf6Sp4SLdvnuMa2sVNwHBboS7EJkm57XQPVU3/QpyNLHbWDdzwtrlS+ez30S3AdYhLKEOxAG8weOnyrtLJAUen9mTkol8oII1edf7mWWbWVf0nBmly21+nZcmCTISQBtdcyPaEno7fFQMDD26/s0lfKob4Kw8H
vs-ssh.visualstudio.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7Hr1oTWqNqOlzGJOfGJ4NakVyIzf1rXYd4d7wo6jBlkLvCA4odBlL0mDUyZ0/QUfTTqeu+tm22gOsv+VrVTMk6vwRU75gY/y9ut5Mb3bR5BV58dKXyq9A9UeB5Cakehn5Zgm6x1mKoVyf+FFn26iYqXJRgzIZZcZ5V6hrE0Qg39kZm4az48o0AUbf6Sp4SLdvnuMa2sVNwHBboS7EJkm57XQPVU3/QpyNLHbWDdzwtrlS+ez30S3AdYhLKEOxAG8weOnyrtLJAUen9mTkol8oII1edf7mWWbWVf0nBmly21+nZcmCTISQBtdcyPaEno7fFQMDD26/s0lfKob4Kw8H
      EOF
  }
}

data "akp_instance" "test" {
  name = akp_instance.test.name
}`, name, getInstanceVersion())
}

func testAccInstanceResourceConfigSpecFeatures(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "Consolidated test: spec features"
      instance_spec = {
        declarative_management_enabled = true
        extensions = [
          {
            id      = "test-extension"
            version = "v0.1.0"
          }
        ]
        cluster_customization_defaults = {
          auto_upgrade_disabled    = true
          kustomization            = ""
          app_replication          = true
          redis_tunneling          = true
          server_side_diff_enabled = true
        }
        application_set_extension = {
          enabled = true
        }
        appset_policy = {
          policy          = "create-update"
          override_policy = true
        }
        agent_permissions_rules = [
          {
            api_groups = ["*"]
            resources  = ["secrets"]
            verbs      = ["get", "list"]
          }
        ]
        app_in_any_namespace_config = {
          enabled = true
        }
        appset_plugins = [
          {
            name            = "plugin-test"
            token           = "random-token"
            base_url        = "https://example.com"
            request_timeout = 30
          }
        ]
        manifest_generation = {
          kustomize = {
            default_version     = "v5.4.3"
            additional_versions = ["v5.6.0", "v5.7.0"]
          }
        }
      }
    }
  }
}`, name, getInstanceVersion())
}

func testAccInstanceResourceConfigMiscFeatures(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "Consolidated test: misc features"
      instance_spec = {
        declarative_management_enabled = true
        host_aliases = [
          {
            ip        = "192.168.1.100"
            hostnames = ["git.internal.example.com", "registry.internal.example.com"]
          }
        ]
        crossplane_extension = {
          resources = [
            {
              group = "*.crossplane.io"
            }
          ]
        }
        cluster_addons_extension = {
          enabled           = true
          allowed_usernames = ["cluster-admin"]
          allowed_groups    = ["platform-team"]
        }
        manifest_generation = {
          kustomize = {
            default_version = "v5.4.3"
          }
        }
      }
    }
  }
  config_management_plugins = {
    "test-plugin" = {
      enabled = true
      image   = "busybox:latest"
      spec = {
        version = "v1.0"
        generate = {
          command = ["sh"]
          args    = ["-c", "echo '{}'"]
        }
      }
    }
  }
}`, name, getInstanceVersion())
}

func testAccInstanceResourceConfigCMP(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "CMP acceptance test"
      instance_spec = {
        declarative_management_enabled = true
        manifest_generation = {
          kustomize = {
            default_version = "v5.4.3"
          }
        }
      }
    }
  }
  config_management_plugins = {
    "my-plugin" = {
      enabled = true
      image   = "busybox:latest"
      spec = {
        version           = "v1.0"
        preserve_file_mode = true
        init = {
          command = ["sh"]
          args    = ["-c", "echo init"]
        }
        generate = {
          command = ["sh"]
          args    = ["-c", "cat manifest.yaml"]
        }
        discover = {
          file_name = "manifest.yaml"
        }
      }
    }
  }
}`, name, getInstanceVersion())
}

func testAccInstanceResourceConfigCMPUpdate(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "CMP acceptance test"
      instance_spec = {
        declarative_management_enabled = true
        manifest_generation = {
          kustomize = {
            default_version = "v5.4.3"
          }
        }
      }
    }
  }
  config_management_plugins = {
    "my-plugin" = {
      enabled = true
      image   = "alpine:latest"
      spec = {
        version = "v2.0"
        generate = {
          command = ["sh"]
          args    = ["-c", "echo updated"]
        }
      }
    }
  }
}`, name, getInstanceVersion())
}

func runInstance_MinimalSpecImport(t *testing.T) {
	name := acctest.RandomWithPrefix("instance-minimal-spec")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccInstanceMinimalSpecConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_instance.test", "id"),
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.declarative_management_enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.image_updater_enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.audit_extension_enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.sync_history_extension_enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.multi_cluster_k8s_dashboard_enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.backend_ip_allow_list_enabled", "true"),
				),
			},
			{
				Config: providerConfig + testAccInstanceMinimalSpecConfig(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			testAccInstanceImportStateStep(name, testAccInstanceMinimalImportStateVerifyIgnore...),
		},
	})
}

func testAccInstanceMinimalSpecConfig(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "Minimal spec import test"
      instance_spec = {
        declarative_management_enabled     = true
        image_updater_enabled              = true
        backend_ip_allow_list_enabled      = true
        audit_extension_enabled            = true
        sync_history_extension_enabled     = true
        multi_cluster_k8s_dashboard_enabled = true
      }
    }
  }
}`, name, getInstanceVersion())
}

func runInstance_PartialInstanceSpecImport(t *testing.T) {
	name := acctest.RandomWithPrefix("instance-partial-spec")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccInstancePartialInstanceSpecConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_instance.test", "id"),
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.declarative_management_enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.image_updater_enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.audit_extension_enabled", "true"),
				),
			},
			{
				Config: providerConfig + testAccInstancePartialInstanceSpecConfig(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			testAccInstanceImportStateStep(name, testAccInstancePartialSpecImportStateVerifyIgnore...),
		},
	})
}

func testAccInstancePartialInstanceSpecConfig(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "Partial instance spec import test"
      instance_spec = {
        declarative_management_enabled     = true
        image_updater_enabled              = true
        backend_ip_allow_list_enabled      = true
        audit_extension_enabled            = true
        sync_history_extension_enabled     = true
        multi_cluster_k8s_dashboard_enabled = true

        kube_vision_config = {
          ai_config = {
            argocd_slack_service  = "argo-notifications"
            argocd_slack_channels = ["alerts"]
            runbooks = [
              {
                name    = "oom-killed-runbook"
                content = "Steps to handle OOMKilled containers"
              },
              {
                name    = "stuck-sync-runbook"
                content = "Steps to handle stuck syncs"
                applied_to = {
                  argocd_applications = ["guestbook-*"]
                  clusters            = ["prod-cluster"]
                }
              }
            ]
            incidents = {
              triggers = [
                {
                  k8s_namespaces = ["production"]
                  degraded_for   = "10m"
                }
              ]
              webhooks = [
                {
                  name             = "slack-alert"
                  description_path = "{.body.alerts[0].annotations.description}"
                  cluster_path     = "{.query.clusterName}"
                  k8s_namespace_path = "{.body.alerts[0].labels.namespace}"
                  argocd_application_namespace_path = "{.body.alerts[0].labels.app_namespace}"
                }
              ]
            }
          }
        }
      }
    }
  }
}`, name, getInstanceVersion())
}

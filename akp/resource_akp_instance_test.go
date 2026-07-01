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

// labeledStep pairs a human-readable label with a TestStep. numberedSteps uses
// the label to auto-inject a "STEP N/total" banner into the step's PreConfig,
// so step numbering stays correct as steps are added, removed, or reordered.
// Entries with an empty label are passed through unchanged — use them for
// helper steps (e.g. testAccInstanceImportStateStep) that should not appear
// in the banner numbering.
type labeledStep struct {
	label string
	step  resource.TestStep
}

func numberedSteps(entries []labeledStep) []resource.TestStep {
	total := 0
	for _, e := range entries {
		if e.label != "" {
			total++
		}
	}
	out := make([]resource.TestStep, 0, len(entries))
	idx := 0
	for _, e := range entries {
		if e.label == "" {
			out = append(out, e.step)
			continue
		}
		idx++
		label, n, t := e.label, idx, total
		original := e.step.PreConfig
		e.step.PreConfig = func() {
			fmt.Fprintf(os.Stderr, "\n==== STEP %d/%d: %s ====\n", n, t, label)
			if original != nil {
				original()
			}
		}
		out = append(out, e.step)
	}
	return out
}

func runInstanceConfigTests(t *testing.T) {
	name := getInstanceName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: numberedSteps([]labeledStep{
			{label: "Import the shared instance", step: resource.TestStep{
				Config:            providerConfig + testAccInstanceImportConfig(name),
				ImportState:       true,
				ImportStateId:     name,
				ResourceName:      "akp_instance.test",
				ImportStateVerify: false,
			}},
			{label: "AI Config Full", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceAIConfigFull(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.akuity_intelligence_extension.enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_channels.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbook_repos.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbook_repos.0.repo_url", "https://github.com/akuity/akuity-intelligence-examples"),
				),
			}},
			{label: "AI Config Updated", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceAIConfigUpdated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.#", "3"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_channels.#", "3"),
				),
			}},
			{label: "AI Config Minimal", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceAIConfigMinimal(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.name", "basic-runbook"),
				),
			}},
			{label: "AI Config Minimal With Slack", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceAIConfigMinimalWithSlack(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.slack_channel_names.#", "1"),
				),
			}},
			{label: "Incidents Config", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceIncidentsConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.0.title_path", "$.title"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.investigation_approval.scopes.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.investigation_approval.scopes.0.argocd_applications.0", "guestbook-prod"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.investigation_approval.scopes.0.consecutive_auto_closures", "3"),
				),
			}},
			{label: "Metrics Ingress Password Hash", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceMetricsIngressPasswordHash(name, "test-bcrypt-hash-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_username", "metrics-user"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "test-bcrypt-hash-1"),
				),
			}},
			{label: "Metrics Ingress Password Hash Updated Description", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceMetricsIngressPasswordHashUpdatedDescription(name, "test-bcrypt-hash-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.description", "Updated description"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "test-bcrypt-hash-1"),
				),
			}},
			{label: "Update password hash", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceMetricsIngressPasswordHashUpdatedDescription(name, "test-bcrypt-hash-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "test-bcrypt-hash-2"),
				),
			}},
			{label: "AI Config with Incidents and Runbooks", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceAIConfigWithIncidentsAndRunbooks(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.argocd_slack_service", "argo-notifications"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.triggers.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "bcrypt-hashed-password"),
				),
			}},
			{step: testAccInstanceImportStateStep(name, testAccInstanceMetricsImportStateVerifyIgnore...)},
			{label: "Re-apply same config", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceAIConfigWithIncidentsAndRunbooks(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.metrics_ingress_password_hash", "bcrypt-hashed-password"),
				),
			}},
			{label: "Core Fields", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceConfigCoreFields(name),
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
			}},
			{step: testAccInstanceImportStateStep(name, testAccInstanceCoreFieldsImportStateVerifyIgnore...)},
			{label: "Spec Features", step: resource.TestStep{
				PreConfig: func() { time.Sleep(30 * time.Second) },
				Config:    providerConfig + testAccInstanceResourceConfigSpecFeatures(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.extensions.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.cluster_customization_defaults.auto_upgrade_disabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.appset_policy.policy", "create-update"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.manifest_generation.kustomize.default_version", "v5.4.3"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.manifest_generation.kustomize.additional_versions.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.termination_protection_enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.termination_protection_notes", "Critical production instance - do not delete"),
				),
			}},
			{label: "Misc Features", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceConfigMiscFeatures(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.host_aliases.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.crossplane_extension.resources.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "config_management_plugins.test-plugin.enabled", "true"),
				),
			}},
			{label: "Secrets Sync", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceConfigSecretsSync(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Sources: two entries, mixed selector shapes
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.#", "2"),
					// sources[0].clusters: match_labels + match_expressions on the same selector
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.clusters.match_labels.role", "secret-source"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.clusters.match_labels.region", "us-east-1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.clusters.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.clusters.match_expressions.0.key", "tier"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.clusters.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.clusters.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.clusters.match_expressions.0.values.0", "primary"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.clusters.match_expressions.0.values.1", "secondary"),
					// sources[0].secrets: multiple expressions with different operators
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.secrets.match_labels.app", "shared"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.secrets.match_expressions.#", "2"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.secrets.match_expressions.0.key", "akuity.io/secret-sync"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.secrets.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.secrets.match_expressions.0.values.0", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.secrets.match_expressions.1.key", "env"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.secrets.match_expressions.1.operator", "NotIn"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.secrets.match_expressions.1.values.0", "dev"),
					// sources[1]: simpler shape, exercises the second list element
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.1.clusters.match_labels.role", "backup-source"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.secrets.sources.1.secrets.match_expressions.0.key", "akuity.io/backup"),
					// Data source: verify sources and match_expressions paths round-trip on read
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.secrets.sources.#", "2"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.clusters.match_labels.role", "secret-source"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.secrets.match_expressions.1.operator", "NotIn"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.secrets.sources.0.secrets.match_expressions.0.values.0", "true"),
				),
			}},
			{label: "Secrets Sync re-apply (no drift)", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceConfigSecretsSync(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			}},
			{label: "CMP Create", step: resource.TestStep{
				PreConfig: func() { time.Sleep(30 * time.Second) },
				Config:    providerConfig + testAccInstanceResourceConfigCMP(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "config_management_plugins.my-plugin.enabled", "true"),
					resource.TestCheckResourceAttr("akp_instance.test", "config_management_plugins.my-plugin.image", "busybox:latest"),
					resource.TestCheckResourceAttr("akp_instance.test", "config_management_plugins.my-plugin.spec.version", "v1.0"),
				),
			}},
			{label: "CMP Update", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceConfigCMPUpdate(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "config_management_plugins.my-plugin.image", "alpine:latest"),
					resource.TestCheckResourceAttr("akp_instance.test", "config_management_plugins.my-plugin.spec.version", "v2.0"),
				),
			}},
			{label: "IgnoreResourceUpdates config", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceIgnoreResourceUpdates(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_cm.resource.ignoreResourceUpdatesEnabled", "true"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_cm.resource.customizations.ignoreResourceUpdates.all"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_cm.resource.customizations.ignoreResourceUpdates.argoproj.io_Application"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_cm.resource.customizations.ignoreDifferences.admissionregistration.k8s.io_MutatingWebhookConfiguration"),
				),
			}},
			{label: "Re-apply ignoreResourceUpdates (no drift)", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceIgnoreResourceUpdates(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			}},
			{label: "Combined resource.customizations with quoted YAML keys", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceCombinedResourceCustomizations(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_cm.resource.customizations"),
				),
			}},
			{label: "Re-apply combined resource.customizations (no drift)", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceCombinedResourceCustomizations(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			}},
			{label: "Modify unrelated field with combined customizations in state", step: resource.TestStep{
				// Regression test for https://github.com/akuityio/akuity-platform/issues/11210:
				// The portal API splits non-wildcard entries of a combined `resource.customizations`
				// YAML back into individual `resource.customizations.<resource>.<group>_<kind>` keys
				// on export. Without filtering in ToConfigMapTFModel, those individual keys leak
				// into state alongside the planned combined value. SuppressNonConfigKeys then
				// preserves both forms in any plan that modifies an unrelated field, and apply
				// fails with: `rpc error: code = InvalidArgument desc = duplicate resources not
				// allowed. group: argoproj.io, kind: Application` because the portal parses the
				// same group/kind out of each form.
				Config: providerConfig + testAccInstanceResourceCombinedResourceCustomizationsWithDescription(name, "Updated description for duplicate resources regression"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.description", "Updated description for duplicate resources regression"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_cm.resource.customizations"),
				),
			}},
			{label: "Remove customizations", step: resource.TestStep{
				Config: providerConfig + testAccInstanceResourceNoCustomizations(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
				),
			}},
		}),
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

// runInstance_RBACChangeWithCombinedCustomizations reproduces the customer-reported
// failure mode for the strip-on-refresh logic in `ToConfigMapTFModel`: an apply
// rejected by the portal with
// `rpc error: code = InvalidArgument desc = duplicate resources not allowed.
//
//	group: <group>, kind: <kind>`
//
// when state has leaked both the combined `resource.customizations` YAML and the
// individual `resource.customizations.<field>.<group>_<kind>` keys derived from
// it by the portal on export.
//
// Putting state into that "leaked both forms" shape against the live portal
// requires a provider whose Read path does not filter the derived keys back out.
// v0.11.0 fits: its `instanceCreateOrUpdate` runs `FilterMapToPlannedKeys` after
// Upsert (so a fresh Apply ends with clean state), but its `instanceRead` does
// not — any Refresh writes the derived keys back into the state file. The test
// drives that explicitly:
//
//  1. Apply against v0.11.0 — state ends clean (filtered post-Apply).
//  2. `RefreshState: true` against v0.11.0 — Refresh runs without the
//     post-Upsert filter, so the state on disk now contains both
//     `argocd_cm.resource.customizations` AND
//     `argocd_cm.resource.customizations.health.argoproj.io_Application`.
//  3. Switch to the in-tree provider and mutate `argocd_rbac_cm.policy.csv`.
//     With the prior `inOld`-guarded #11220 strip, `oldElems` already contains
//     the derived key, the strip is skipped, `SuppressNonConfigKeys` carries the
//     duplicate forward, and Apply sends both forms — portal rejects with
//     `duplicate resources not allowed`. With the unconditional strip, Refresh
//     self-heals and the Apply succeeds.
//  4. Re-apply to assert no perpetual drift.
func runInstance_RBACChangeWithCombinedCustomizations(t *testing.T) {
	name := acctest.RandomWithPrefix("instance-rbac-combined")
	policyV1 := "p, role:org-admin, applications, *, */*, allow\n" +
		"g, your-github-org:your-team, role:org-admin\n"
	policyV2 := policyV1 +
		"p, role:org-admin, clusters, get, *, allow\n"

	// v0.11.0 is intentionally pinned (not `~>`): it has #11193's planned-YAML
	// preservation so step 1's Create completes without an "inconsistent result"
	// error, but no strip logic at all, so state reliably ends with both the
	// combined `resource.customizations` and the derived
	// `resource.customizations.health.argoproj.io_Application` individual key.
	// v0.11.1 already adds the buggy `inOld`-guarded #11220 strip; on a fresh
	// Create its `oldElems` is empty for the individual key, the strip fires,
	// state ends clean, and the regression we want to exercise never sets up.
	leakingProvider := map[string]resource.ExternalProvider{
		"akp": {
			Source:            "akuity/akp",
			VersionConstraint: "= 0.11.0",
		},
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		// Providers are declared per-step: step 1 pins to a published v0.10.x
		// release via `ExternalProviders`, steps 2+ use the in-tree factory.
		// `terraform-plugin-testing` rejects mixing case-level and step-level
		// provider declarations, so the case-level factory is intentionally
		// omitted here.
		Steps: numberedSteps([]labeledStep{
			// Create against the published v0.11.0 provider. Upsert runs
			// FilterMapToPlannedKeys after the API call, so state at this
			// point is filtered (combined only).
			{label: "Create with v0.11.0 provider", step: resource.TestStep{
				ExternalProviders: leakingProvider,
				Config:            providerConfig + testAccInstanceResourceCombinedCustomizationsWithRBAC(name, policyV1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", name),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_cm.resource.customizations"),
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_rbac_cm.policy.csv", policyV1),
				),
			}},
			// Re-apply the same config against v0.11.0. `terraform apply`
			// implicitly refreshes the state first; v0.11.0's instanceRead
			// has no FilterMapToPlannedKeys, so the refresh writes every key
			// the portal returns (including the derived individual ones) into
			// the state. The plan compares that refreshed state to config:
			// SuppressNonConfigKeys preserves state values for keys absent
			// from config, so the plan is empty and no Upsert (and therefore
			// no post-Upsert filter) runs — leaving the state on disk in the
			// "stale, both forms leaked" shape we want. RefreshState: true
			// isn't usable here because the framework rejects pairing it with
			// a Config block (which we still need to materialize the
			// provider configuration for the akp provider).
			{label: "Re-apply with v0.11.0 to leak derived keys into state", step: resource.TestStep{
				ExternalProviders: leakingProvider,
				Config:            providerConfig + testAccInstanceResourceCombinedCustomizationsWithRBAC(name, policyV1),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Confirm the leak landed; if this stops being true the
					// downstream assertions are no longer meaningful.
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_cm.resource.customizations.health.argoproj.io_Application"),
				),
			}},
			// Switch to the in-tree provider and mutate only
			// `argocd_rbac_cm.policy.csv`. With the `inOld`-guarded strip
			// (i.e. the regression), the in-tree refresh sees the leaked
			// individual key in oldElems, the strip is skipped, the duplicate
			// is carried into the plan by SuppressNonConfigKeys, and the
			// Update sends both forms → portal rejects with
			// `duplicate resources not allowed`. With the unconditional strip,
			// refresh self-heals and the apply succeeds.
			{label: "Mutate RBAC csv with in-tree provider (the customer-reported failing apply)", step: resource.TestStep{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   providerConfig + testAccInstanceResourceCombinedCustomizationsWithRBAC(name, policyV2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd_rbac_cm.policy.csv", policyV2),
					resource.TestCheckResourceAttrSet("akp_instance.test", "argocd_cm.resource.customizations"),
					resource.TestCheckNoResourceAttr("akp_instance.test", "argocd_cm.resource.customizations.health.argoproj.io_Application"),
				),
			}},
			{label: "Re-apply after RBAC change (no drift)", step: resource.TestStep{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   providerConfig + testAccInstanceResourceCombinedCustomizationsWithRBAC(name, policyV2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			}},
		}),
	})
}

func testAccInstanceResourceCombinedCustomizationsWithRBAC(name, policyCSV string) string {
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
    "resource.customizations" = <<-EOF
      'argoproj.io/Application':
        health.lua: |
          hs = {}
          hs.status = "Healthy"
          hs.message = "Healthy"
          return hs
      '*.crossplane.io/*':
        health.lua: |
          hs = {}
          hs.status = "Healthy"
          hs.message = "Resource is up-to-date."
          return hs
    EOF
  }
  argocd_rbac_cm = {
    "policy.default" = "role:readonly"
    "policy.csv"     = %q
  }
}
`, name, getInstanceVersion(), policyCSV)
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

// testAccInstanceResourceCombinedResourceCustomizations returns a config that uses the
// combined `resource.customizations` argocd_cm key with single-quoted YAML keys.
// Regression test for https://github.com/akuityio/akuity-platform/issues/11183: the
// portal API parses the YAML into typed structs and re-serializes it on export, which
// drops optional quoting around keys like `'argoproj.io/Application'`. Without the fix
// in ToConfigMapTFModel, this triggers "Provider produced inconsistent result after
// apply" because the planned (quoted) value differs from the API-returned (unquoted) one.
func testAccInstanceResourceCombinedResourceCustomizations(name string) string {
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
    "resource.customizations" = <<-EOF
      'argoproj.io/Application':
        health.lua: |
          hs = {}
          hs.status = "Healthy"
          hs.message = "Healthy"
          return hs
      '*.crossplane.io/*':
        health.lua: |
          hs = {}
          hs.status = "Healthy"
          hs.message = "Resource is up-to-date."
          return hs
    EOF
  }
}
`, name, getInstanceVersion())
}

// testAccInstanceResourceCombinedResourceCustomizationsWithDescription returns the same
// combined `resource.customizations` config as testAccInstanceResourceCombinedResourceCustomizations
// but with a configurable description so we can force a plan diff outside `argocd_cm`.
// The plan modifier SuppressNonConfigKeys preserves the (post-refresh) state value of
// argocd_cm verbatim when config is a subset of state; pairing that with a forced diff
// elsewhere is what makes Update fire and surfaces the "duplicate resources not allowed"
// API error when state has leaked both combined and individual forms.
func testAccInstanceResourceCombinedResourceCustomizationsWithDescription(name, description string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      description = %q
      version     = %q
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
    "resource.customizations" = <<-EOF
      'argoproj.io/Application':
        health.lua: |
          hs = {}
          hs.status = "Healthy"
          hs.message = "Healthy"
          return hs
      '*.crossplane.io/*':
        health.lua: |
          hs = {}
          hs.status = "Healthy"
          hs.message = "Resource is up-to-date."
          return hs
    EOF
  }
}
`, name, description, getInstanceVersion())
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
            runbook_repos = [
              {
                repo_url = "https://github.com/akuity/akuity-intelligence-examples"
                revision = "main"
                path     = "runbooks"
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
                  title_path                          = "$.title"
                }
              ]
              investigation_approval = {
                scopes = [
                  {
                    argocd_applications       = ["guestbook-prod"]
                    k8s_namespaces            = ["production"]
                    clusters                  = ["prod-cluster"]
                    consecutive_auto_closures = 3
                  }
                ]
              }
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
        termination_protection_enabled = true
        termination_protection_notes   = "Critical production instance - do not delete"
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
        declarative_management_enabled  = true
        termination_protection_enabled  = false
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

func testAccInstanceResourceConfigSecretsSync(name string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = %q
  argocd = {
    spec = {
      version     = %q
      description = "Consolidated test: secrets sync"
      instance_spec = {
        declarative_management_enabled = true
        secrets = {
          sources = [
            {
              clusters = {
                match_labels = {
                  role   = "secret-source"
                  region = "us-east-1"
                }
                match_expressions = [
                  {
                    key      = "tier"
                    operator = "In"
                    values   = ["primary", "secondary"]
                  }
                ]
              }
              secrets = {
                match_labels = {
                  app = "shared"
                }
                match_expressions = [
                  {
                    key      = "akuity.io/secret-sync"
                    operator = "In"
                    values   = ["true"]
                  },
                  {
                    key      = "env"
                    operator = "NotIn"
                    values   = ["dev"]
                  }
                ]
              }
            },
            {
              clusters = {
                match_labels = {
                  role = "backup-source"
                }
              }
              secrets = {
                match_expressions = [
                  {
                    key      = "akuity.io/backup"
                    operator = "In"
                    values   = ["enabled"]
                  }
                ]
              }
            }
          ]
        }
      }
    }
  }
}
data "akp_instance" "test" {
  name = akp_instance.test.name
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

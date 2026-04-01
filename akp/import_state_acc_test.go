//go:build !unit

package akp

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var (
	// Cluster common: SuppressProtobufDefault handles plan diffs for real users,
	// but ImportStateVerify compares raw state values bypassing plan modifiers.
	testAccClusterCommonImportStateVerifyIgnore = []string{
		"spec.data.app_replication",                     // SuppressProtobufDefault (bool)
		"spec.data.auto_upgrade_disabled",               // SuppressProtobufDefault (bool)
		"spec.data.kustomization",                       // UnknownWhenCustomSize custom modifier
		"spec.data.maintenance_mode",                    // SuppressProtobufDefault (bool); prefix also covers maintenance_mode_expiry
		"spec.data.multi_cluster_k8s_dashboard_enabled", // SuppressProtobufDefault (bool)
		"spec.data.pod_inherit_metadata",                // SuppressProtobufDefault (bool)
		"spec.data.project",                             // Optional-only (no Computed)
		"spec.data.redis_tunneling",                     // SuppressProtobufDefault (bool)
		"spec.data.server_side_diff_enabled",            // SuppressProtobufDefault (bool)
		"spec.namespace_scoped",                         // Optional-only (no Computed)
	}
	testAccClusterKustomizationImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccClusterCommonImportStateVerifyIgnore,
		"spec.data.kustomization",
	)
	testAccClusterCustomSizeImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccClusterCommonImportStateVerifyIgnore,
		"spec.data.custom_agent_size_config",
		"spec.data.kustomization",
		"spec.data.size",
	)
	testAccClusterAutoSizeImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccClusterCommonImportStateVerifyIgnore,
		"spec.data.auto_agent_size_config",
	)
	testAccClusterManagedConfigImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccClusterCommonImportStateVerifyIgnore,
		"spec.data.managed_cluster_config",
	)
	testAccClusterNotificationsImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccClusterCommonImportStateVerifyIgnore,
		"spec.data.argocd_notifications_settings",
	)
	testAccClusterCompatibilityImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccClusterCommonImportStateVerifyIgnore,
		"spec.data.compatibility",
	)
	testAccClusterProjectImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccClusterCommonImportStateVerifyIgnore,
		"spec.data.project",
	)
	testAccClusterNamespaceScopedFalseImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccClusterCommonImportStateVerifyIgnore,
		"spec.namespace_scoped",
	)
	testAccClusterReapplyImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccClusterCommonImportStateVerifyIgnore,
		"reapply_manifests_on_update",
	)
	testAccClusterFeaturesImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccClusterCommonImportStateVerifyIgnore,
		"spec.data.datadog_annotations_enabled",
		"spec.data.eks_addon_enabled",
	)
	testAccKargoAgentCommonImportStateVerifyIgnore = []string{
		"spec.data.auto_upgrade_disabled", // SuppressProtobufDefault (bool)
		"spec.data.allowed_job_sa",        // Optional-only (no Computed)
		"spec.data.pod_inherit_metadata",  // SuppressProtobufDefault (bool)
	}
	testAccKargoAgentKustomizationImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccKargoAgentCommonImportStateVerifyIgnore,
		"spec.data.kustomization", // SuppressProtobufDefault (string)
	)
	testAccKargoAgentAllowedJobSAImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccKargoAgentCommonImportStateVerifyIgnore,
	)
	testAccKargoAgentReapplyImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccKargoAgentCommonImportStateVerifyIgnore,
		"reapply_manifests_on_update", // SuppressProtobufDefault (bool)
	)
	// Instance common: SuppressProtobufDefault handles plan diffs for real users,
	// but ImportStateVerify compares raw state values bypassing plan modifiers.
	testAccInstanceCommonImportStateVerifyIgnore = []string{
		// UseStateForNullUnknown objects
		"argocd.spec.instance_spec.akuity_intelligence_extension",                      // UseStateForNullUnknown
		"argocd.spec.instance_spec.appset_policy",                                      // UseStateForNullUnknown
		"argocd.spec.instance_spec.cluster_customization_defaults",                     // UseStateForNullUnknown
		"argocd.spec.instance_spec.kube_vision_config.cve_scan_config",                 // UseStateForNullUnknown
		"argocd.spec.instance_spec.kube_vision_config.cve_scan_config.%",               // internal marker
		"argocd.spec.instance_spec.kube_vision_config.cve_scan_config.rescan_interval", // Optional-only inside UseStateForNullUnknown
		"argocd.spec.instance_spec.kube_vision_config.cve_scan_config.scan_enabled",    // Optional-only inside UseStateForNullUnknown
		// SuppressProtobufDefault fields (ImportStateVerify bypasses plan modifiers)
		"argocd.spec.instance_spec.assistant_extension_enabled",         // SuppressProtobufDefault (bool)
		"argocd.spec.instance_spec.audit_extension_enabled",             // SuppressProtobufDefault (bool)
		"argocd.spec.instance_spec.backend_ip_allow_list_enabled",       // SuppressProtobufDefault (bool)
		"argocd.spec.instance_spec.fqdn",                                // SuppressProtobufDefault (string)
		"argocd.spec.instance_spec.image_updater_enabled",               // SuppressProtobufDefault (bool)
		"argocd.spec.instance_spec.metrics_ingress_username",            // SuppressProtobufDefault (string)
		"argocd.spec.instance_spec.multi_cluster_k8s_dashboard_enabled", // SuppressProtobufDefault (bool)
		"argocd.spec.instance_spec.privileged_notification_cluster",     // SuppressProtobufDefault (string)
		"argocd.spec.instance_spec.sync_history_extension_enabled",      // SuppressProtobufDefault (bool)
		// Optional-only fields
		"argocd.spec.instance_spec.appset_plugins", // Optional-only
		"argocd.spec.instance_spec.host_aliases",   // Optional-only
		"argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.webhooks.0.argocd_application_name_path", // Optional-only
		"argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to.degraded_for",                // Optional-only
		"argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.1.applied_to.degraded_for",                // Optional-only
		"argocd_resources", // Optional-only
		// Custom modifiers
		"argocd_cm",                 // SuppressNonConfigKeys
		"config_management_plugins", // SuppressProtobufDefault on nested version
	}
	testAccInstanceMetricsImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccInstanceCommonImportStateVerifyIgnore,
		"argocd.spec.instance_spec.metrics_ingress_password_hash",
	)
	testAccInstanceCoreFieldsImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccInstanceCommonImportStateVerifyIgnore,
		"argocd_secret",
		"argocd.spec.instance_spec.metrics_ingress_password_hash",
		"application_set_secret",
		"argocd_notifications_secret",
		"argocd_image_updater_secret",
		"repo_credential_secrets",
		"repo_template_credential_secrets",
	)
	// Kargo common: SuppressProtobufDefault handles plan diffs for real users,
	// but ImportStateVerify compares raw state values bypassing plan modifiers.
	testAccKargoInstanceCommonImportStateVerifyIgnore = []string{
		// UseStateForNullUnknown objects
		"kargo.spec.kargo_instance_spec.akuity_intelligence", // UseStateForNullUnknown
		// SuppressProtobufDefault fields (ImportStateVerify bypasses plan modifiers)
		"kargo.spec.fqdn", // SuppressProtobufDefault (string)
		"kargo.spec.kargo_instance_spec.promo_controller_enabled", // SuppressProtobufDefault (bool)
		"kargo.spec.oidc_config.cli_client_id",                    // SuppressProtobufDefault (string)
		"kargo.spec.oidc_config.client_id",                        // SuppressProtobufDefault (string)
		"kargo.spec.oidc_config.dex_config",                       // SuppressProtobufDefault (string)
		"kargo.spec.oidc_config.issuer_url",                       // SuppressProtobufDefault (string)
		"workspace",                                               // SuppressProtobufDefault (string)
		// Optional-only fields
		"kargo.spec.kargo_instance_spec.argocd_ui",                 // Optional-only
		"kargo.spec.kargo_instance_spec.global_credentials_ns",     // Optional-only
		"kargo.spec.kargo_instance_spec.global_service_account_ns", // Optional-only
		"kargo.spec.kargo_instance_spec.ip_allow_list",             // Optional-only
		"kargo.spec.oidc_config.additional_scopes",                 // Optional-only
		"kargo_resources", // Optional-only
	}
	testAccKargoDexConfigSecretImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccKargoInstanceCommonImportStateVerifyIgnore,
		"kargo.spec.oidc_config.dex_config_secret",
	)
	testAccKargoSecretImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccKargoInstanceCommonImportStateVerifyIgnore,
		"kargo_secret",
	)
	testAccInstanceIPAllowListImportStateVerifyIgnore = []string{
		"id",
	}

	// New import ignore lists derived per-config. SuppressProtobufDefault handles
	// plan diffs for real users, but ImportStateVerify bypasses plan modifiers and
	// compares raw state values, so these fields must still be listed here.

	// --- Cluster ---

	testAccClusterMinimalImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccClusterCommonImportStateVerifyIgnore,
		"spec.data.datadog_annotations_enabled",   // Optional-only, omitted
		"spec.data.eks_addon_enabled",             // Optional-only, omitted
		"spec.data.compatibility",                 // UseStateForNullUnknown, omitted
		"spec.data.argocd_notifications_settings", // UseStateForNullUnknown, omitted
	)
	testAccClusterPartialCustomSizeImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccClusterMinimalImportStateVerifyIgnore,
		"spec.data.custom_agent_size_config", // Optional-only, partially set
		"spec.data.size",                     // size=custom interaction
	)
	testAccClusterPartialNotificationsImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccClusterCommonImportStateVerifyIgnore,
		"spec.data.datadog_annotations_enabled", // Optional-only, omitted
		"spec.data.eks_addon_enabled",           // Optional-only, omitted
		// compatibility and argocd_notifications_settings are SET in this config — not ignored
	)

	// --- Kargo Agent ---

	testAccKargoAgentMinimalImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccKargoAgentCommonImportStateVerifyIgnore,
	)
	testAccKargoAgentPartialDataImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccKargoAgentCommonImportStateVerifyIgnore,
	)

	// --- Instance ---

	// Minimal: kube_vision_config and all other UseStateForNullUnknown objects are omitted entirely
	testAccInstanceMinimalImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccInstanceCommonImportStateVerifyIgnore,
		"argocd.spec.instance_spec.kube_vision_config",          // UseStateForNullUnknown, omitted
		"argocd.spec.instance_spec.crossplane_extension",        // UseStateForNullUnknown, omitted
		"argocd.spec.instance_spec.manifest_generation",         // UseStateForNullUnknown, omitted
		"argocd.spec.instance_spec.application_set_extension",   // UseStateForNullUnknown, omitted
		"argocd.spec.instance_spec.cluster_addons_extension",    // UseStateForNullUnknown, omitted
		"argocd.spec.instance_spec.app_in_any_namespace_config", // UseStateForNullUnknown, omitted
		"argocd_notifications_cm",                               // UseStateForNullUnknown Map, omitted
		"argocd_image_updater_config",                           // UseStateForNullUnknown Map, omitted
		"argocd_image_updater_ssh_config",                       // UseStateForNullUnknown Map, omitted
	)
	// Partial: kube_vision_config IS SET so not ignored; its omitted children are.
	testAccInstancePartialSpecImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccInstanceCommonImportStateVerifyIgnore,
		"argocd.spec.instance_spec.crossplane_extension",        // UseStateForNullUnknown, omitted
		"argocd.spec.instance_spec.manifest_generation",         // UseStateForNullUnknown, omitted
		"argocd.spec.instance_spec.application_set_extension",   // UseStateForNullUnknown, omitted
		"argocd.spec.instance_spec.cluster_addons_extension",    // UseStateForNullUnknown, omitted
		"argocd.spec.instance_spec.app_in_any_namespace_config", // UseStateForNullUnknown, omitted
		"argocd_notifications_cm",                               // UseStateForNullUnknown Map, omitted
		"argocd_image_updater_config",                           // UseStateForNullUnknown Map, omitted
		"argocd_image_updater_ssh_config",                       // UseStateForNullUnknown Map, omitted
		// UseStateForNullUnknown children inside set kube_vision_config
		"argocd.spec.instance_spec.kube_vision_config.ai_config.runbooks.0.applied_to", // UseStateForNullUnknown, omitted
		"argocd.spec.instance_spec.kube_vision_config.ai_config.incidents.grouping",    // UseStateForNullUnknown, omitted
	)

	// --- Kargo Instance ---

	// Minimal: oidc_config, agent_customization_defaults, akuity_intelligence, gc_config all omitted
	testAccKargoMinimalImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccKargoInstanceCommonImportStateVerifyIgnore,
		"kargo.spec.oidc_config",                                      // UseStateForNullUnknown, omitted
		"kargo.spec.kargo_instance_spec.agent_customization_defaults", // UseStateForNullUnknown, omitted
		"kargo.spec.kargo_instance_spec.gc_config",                    // UseStateForNullUnknown, omitted
	)
	// PartialOIDC: oidc_config IS SET (with admin_account), but other accounts are omitted
	testAccKargoPartialOIDCImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccKargoInstanceCommonImportStateVerifyIgnore,
		"kargo.spec.kargo_instance_spec.agent_customization_defaults", // UseStateForNullUnknown, omitted
		"kargo.spec.kargo_instance_spec.gc_config",                    // UseStateForNullUnknown, omitted
		"kargo.spec.oidc_config.viewer_account",                       // UseStateForNullUnknown, omitted inside set oidc_config
		"kargo.spec.oidc_config.user_account",                         // UseStateForNullUnknown, omitted inside set oidc_config
		"kargo.spec.oidc_config.project_creator_account",              // UseStateForNullUnknown, omitted inside set oidc_config
	)
	// PartialSpec: akuity_intelligence and gc_config are SET, oidc_config and agent_customization_defaults omitted
	testAccKargoPartialSpecImportStateVerifyIgnore = appendImportStateVerifyIgnore(
		testAccKargoInstanceCommonImportStateVerifyIgnore,
		"kargo.spec.oidc_config",                                               // UseStateForNullUnknown, omitted
		"kargo.spec.kargo_instance_spec.agent_customization_defaults",          // UseStateForNullUnknown, omitted
		"kargo.spec.kargo_instance_spec.akuity_intelligence.allowed_usernames", // Optional-only, omitted inside set akuity_intelligence
		"kargo.spec.kargo_instance_spec.akuity_intelligence.allowed_groups",    // Optional-only, omitted inside set akuity_intelligence
		"kargo.spec.kargo_instance_spec.akuity_intelligence.model_version",     // SuppressProtobufDefault (string), inside set parent
	)
)

func testAccClusterImportStateStep(instanceID, name string, ignore ...string) resource.TestStep {
	return resource.TestStep{
		ResourceName:            "akp_cluster.test",
		ImportState:             true,
		ImportStateId:           fmt.Sprintf("%s/%s", instanceID, name),
		ImportStateVerify:       true,
		ImportStateVerifyIgnore: cloneImportStateVerifyIgnore(ignore),
	}
}

func testAccInstanceImportStateStep(name string, ignore ...string) resource.TestStep {
	return resource.TestStep{
		ResourceName:            "akp_instance.test",
		ImportState:             true,
		ImportStateId:           name,
		ImportStateVerify:       true,
		ImportStateVerifyIgnore: cloneImportStateVerifyIgnore(ignore),
	}
}

func testAccKargoImportStateStep(name string, ignore ...string) resource.TestStep {
	return resource.TestStep{
		ResourceName:            "akp_kargo_instance.test",
		ImportState:             true,
		ImportStateId:           name,
		ImportStateVerify:       true,
		ImportStateVerifyIgnore: cloneImportStateVerifyIgnore(ignore),
	}
}

func testAccKargoAgentImportStateStep(instanceID, name string, ignore ...string) resource.TestStep {
	return resource.TestStep{
		ResourceName:            "akp_kargo_agent.test",
		ImportState:             true,
		ImportStateId:           fmt.Sprintf("%s/%s", instanceID, name),
		ImportStateVerify:       true,
		ImportStateVerifyIgnore: cloneImportStateVerifyIgnore(ignore),
	}
}

func testAccInstanceIPAllowListImportStateStep(instanceID string, ignore ...string) resource.TestStep {
	return resource.TestStep{
		ResourceName:                         "akp_instance_ip_allow_list.test",
		ImportState:                          true,
		ImportStateId:                        instanceID,
		ImportStateVerify:                    true,
		ImportStateVerifyIdentifierAttribute: "instance_id",
		ImportStateVerifyIgnore:              cloneImportStateVerifyIgnore(ignore),
	}
}

func cloneImportStateVerifyIgnore(ignore []string) []string {
	return append([]string(nil), ignore...)
}

func appendImportStateVerifyIgnore(base []string, ignore ...string) []string {
	res := cloneImportStateVerifyIgnore(base)
	return append(res, ignore...)
}

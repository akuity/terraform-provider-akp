// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2023 Akuity, Inc.
*/

package types

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ArgoCD struct {
	Spec ArgoCDSpec `tfsdk:"spec"`
}

type ArgoCDSpec struct {
	Description  types.String `tfsdk:"description"`
	Version      types.String `tfsdk:"version"`
	InstanceSpec InstanceSpec `tfsdk:"instance_spec"`
}

type ArgoCDExtensionInstallEntry struct {
	Id      types.String `tfsdk:"id"`
	Version types.String `tfsdk:"version"`
}

type ClusterCustomization struct {
	AutoUpgradeDisabled types.Bool   `tfsdk:"auto_upgrade_disabled"`
	Kustomization       types.String `tfsdk:"kustomization"`
	AppReplication      types.Bool   `tfsdk:"app_replication"`
	RedisTunneling      types.Bool   `tfsdk:"redis_tunneling"`
}

type AppsetPolicy struct {
	Policy         types.String `tfsdk:"policy"`
	OverridePolicy types.Bool   `tfsdk:"override_policy"`
}

type AgentPermissionsRule struct {
	ApiGroups []types.String `tfsdk:"api_groups"`
	Resources []types.String `tfsdk:"resources"`
	Verbs     []types.String `tfsdk:"verbs"`
}

type CrossplaneExtensionResource struct {
	Group types.String `tfsdk:"group"`
}

type CrossplaneExtension struct {
	Resources []*CrossplaneExtensionResource `tfsdk:"resources"`
}

type AkuityIntelligenceExtension struct {
	Enabled                  types.Bool     `tfsdk:"enabled"`
	AllowedUsernames         []types.String `tfsdk:"allowed_usernames"`
	AllowedGroups            []types.String `tfsdk:"allowed_groups"`
	AiSupportEngineerEnabled types.Bool     `tfsdk:"ai_support_engineer_enabled"`
}

type TargetSelector struct {
	ArgocdApplications []types.String `tfsdk:"argocd_applications"`
	K8SNamespaces      []types.String `tfsdk:"k8s_namespaces"`
	Clusters           []types.String `tfsdk:"clusters"`
	DegradedFor        types.String   `tfsdk:"degraded_for"`
}

type Runbook struct {
	Name      types.String    `tfsdk:"name"`
	Content   types.String    `tfsdk:"content"`
	AppliedTo *TargetSelector `tfsdk:"applied_to"`
}

type IncidentWebhookConfig struct {
	Name                      types.String `tfsdk:"name"`
	DescriptionPath           types.String `tfsdk:"description_path"`
	ClusterPath               types.String `tfsdk:"cluster_path"`
	K8SNamespacePath          types.String `tfsdk:"k8s_namespace_path"`
	ArgocdApplicationNamePath types.String `tfsdk:"argocd_application_name_path"`
}

type IncidentsConfig struct {
	Triggers []*TargetSelector        `tfsdk:"triggers"`
	Webhooks []*IncidentWebhookConfig `tfsdk:"webhooks"`
}

type AIConfig struct {
	Runbooks           []*Runbook       `tfsdk:"runbooks"`
	Incidents          *IncidentsConfig `tfsdk:"incidents"`
	ArgocdSlackService types.String     `tfsdk:"argocd_slack_service"`
}

type KubeVisionConfig struct {
	CveScanConfig *CveScanConfig `tfsdk:"cve_scan_config"`
	AiConfig      *AIConfig      `tfsdk:"ai_config"`
}

type AppInAnyNamespaceConfig struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type CustomDeprecatedAPI struct {
	ApiVersion                     types.String `tfsdk:"api_version"`
	NewApiVersion                  types.String `tfsdk:"new_api_version"`
	DeprecatedInKubernetesVersion  types.String `tfsdk:"deprecated_in_kubernetes_version"`
	UnavailableInKubernetesVersion types.String `tfsdk:"unavailable_in_kubernetes_version"`
}

type CveScanConfig struct {
	ScanEnabled    types.Bool   `tfsdk:"scan_enabled"`
	RescanInterval types.String `tfsdk:"rescan_interval"`
}

type ObjectSelector struct {
	MatchLabels      types.Map                   `tfsdk:"match_labels"`
	MatchExpressions []*LabelSelectorRequirement `tfsdk:"match_expressions"`
}

type LabelSelectorRequirement struct {
	Key      types.String   `tfsdk:"key"`
	Operator types.String   `tfsdk:"operator"`
	Values   []types.String `tfsdk:"values"`
}

type ClusterSecretMapping struct {
	Clusters *ObjectSelector `tfsdk:"clusters"`
	Secrets  *ObjectSelector `tfsdk:"secrets"`
}

type SecretsManagementConfig struct {
	Sources      []*ClusterSecretMapping `tfsdk:"sources"`
	Destinations []*ClusterSecretMapping `tfsdk:"destinations"`
}

type ApplicationSetExtension struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type BucketRateLimiting struct {
	Enabled    types.Bool  `tfsdk:"enabled"`
	BucketSize types.Int64 `tfsdk:"bucket_size"`
	BucketQps  types.Int64 `tfsdk:"bucket_qps"`
}

type ItemRateLimiting struct {
	Enabled         types.Bool    `tfsdk:"enabled"`
	FailureCooldown types.Int64   `tfsdk:"failure_cooldown"`
	BaseDelay       types.Int64   `tfsdk:"base_delay"`
	MaxDelay        types.Int64   `tfsdk:"max_delay"`
	BackoffFactor   types.Float64 `tfsdk:"backoff_factor"`
}

type AppReconciliationsRateLimiting struct {
	BucketRateLimiting *BucketRateLimiting `tfsdk:"bucket_rate_limiting"`
	ItemRateLimiting   *ItemRateLimiting   `tfsdk:"item_rate_limiting"`
}

type InstanceSpec struct {
	IpAllowList                     []*IPAllowListEntry          `tfsdk:"ip_allow_list"`
	Subdomain                       types.String                 `tfsdk:"subdomain"`
	DeclarativeManagementEnabled    types.Bool                   `tfsdk:"declarative_management_enabled"`
	Extensions                      types.List                   `tfsdk:"extensions"`
	ClusterCustomizationDefaults    types.Object                 `tfsdk:"cluster_customization_defaults"`
	ImageUpdaterEnabled             types.Bool                   `tfsdk:"image_updater_enabled"`
	BackendIpAllowListEnabled       types.Bool                   `tfsdk:"backend_ip_allow_list_enabled"`
	RepoServerDelegate              *RepoServerDelegate          `tfsdk:"repo_server_delegate"`
	AuditExtensionEnabled           types.Bool                   `tfsdk:"audit_extension_enabled"`
	SyncHistoryExtensionEnabled     types.Bool                   `tfsdk:"sync_history_extension_enabled"`
	CrossplaneExtension             *CrossplaneExtension         `tfsdk:"crossplane_extension"`
	ImageUpdaterDelegate            *ImageUpdaterDelegate        `tfsdk:"image_updater_delegate"`
	AppSetDelegate                  *AppSetDelegate              `tfsdk:"app_set_delegate"`
	AssistantExtensionEnabled       types.Bool                   `tfsdk:"assistant_extension_enabled"`
	AppsetPolicy                    types.Object                 `tfsdk:"appset_policy"`
	HostAliases                     []*HostAliases               `tfsdk:"host_aliases"`
	AgentPermissionsRules           []*AgentPermissionsRule      `tfsdk:"agent_permissions_rules"`
	Fqdn                            types.String                 `tfsdk:"fqdn"`
	MultiClusterK8SDashboardEnabled types.Bool                   `tfsdk:"multi_cluster_k8s_dashboard_enabled"`
	AkuityIntelligenceExtension     *AkuityIntelligenceExtension `tfsdk:"akuity_intelligence_extension"`
	KubeVisionConfig                *KubeVisionConfig            `tfsdk:"kube_vision_config"`
	AppInAnyNamespaceConfig         *AppInAnyNamespaceConfig     `tfsdk:"app_in_any_namespace_config"`
	AppsetPlugins                   []*AppsetPlugins             `tfsdk:"appset_plugins"`
	ApplicationSetExtension         *ApplicationSetExtension     `tfsdk:"application_set_extension"`
}

type AppsetPlugins struct {
	Name           types.String `tfsdk:"name"`
	Token          types.String `tfsdk:"token"`
	BaseUrl        types.String `tfsdk:"base_url"`
	RequestTimeout types.Int64  `tfsdk:"request_timeout"`
}

type ManagedCluster struct {
	ClusterName types.String `tfsdk:"cluster_name"`
}

type RepoServerDelegate struct {
	ControlPlane   types.Bool      `tfsdk:"control_plane"`
	ManagedCluster *ManagedCluster `tfsdk:"managed_cluster"`
}

type ImageUpdaterDelegate struct {
	ControlPlane   types.Bool      `tfsdk:"control_plane"`
	ManagedCluster *ManagedCluster `tfsdk:"managed_cluster"`
}

type AppSetDelegate struct {
	ManagedCluster *ManagedCluster `tfsdk:"managed_cluster"`
}

type IPAllowListEntry struct {
	Ip          types.String `tfsdk:"ip"`
	Description types.String `tfsdk:"description"`
}

type HostAliases struct {
	Ip        types.String   `tfsdk:"ip"`
	Hostnames []types.String `tfsdk:"hostnames"`
}

// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2023 Akuity, Inc.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ArgoCD struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ArgoCDSpec `json:"spec,omitempty"`
}

type ArgoCDSpec struct {
	Description string `json:"description"`
	Version     string `json:"version"`

	InstanceSpec InstanceSpec `json:"instanceSpec,omitempty"`
}

type ArgoCDExtensionInstallEntry struct {
	Id      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
}

type ClusterCustomization struct {
	AutoUpgradeDisabled *bool                `json:"autoUpgradeDisabled,omitempty"`
	Kustomization       runtime.RawExtension `json:"kustomization,omitempty"`
	AppReplication      *bool                `json:"appReplication,omitempty"`
	RedisTunneling      *bool                `json:"redisTunneling,omitempty"`
}

type AppsetPolicy struct {
	Policy         string `json:"policy,omitempty"`
	OverridePolicy *bool  `json:"overridePolicy,omitempty"`
}

type AgentPermissionsRule struct {
	ApiGroups []string `json:"apiGroups,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Verbs     []string `json:"verbs,omitempty"`
}

type CrossplaneExtensionResource struct {
	Group string `json:"group,omitempty"`
}

type CrossplaneExtension struct {
	Resources []*CrossplaneExtensionResource `json:"resources,omitempty"`
}

type AkuityIntelligenceExtension struct {
	Enabled                  *bool    `json:"enabled,omitempty"`
	AllowedUsernames         []string `json:"allowedUsernames,omitempty"`
	AllowedGroups            []string `json:"allowedGroups,omitempty"`
	AiSupportEngineerEnabled *bool    `json:"aiSupportEngineerEnabled,omitempty"`
	ModelVersion             string   `json:"modelVersion,omitempty"`
}

type TargetSelector struct {
	ArgocdApplications []string `json:"argocdApplications,omitempty"`
	K8SNamespaces      []string `json:"k8sNamespaces,omitempty"`
	Clusters           []string `json:"clusters,omitempty"`
}

type Runbook struct {
	Name      string          `json:"name,omitempty"`
	Content   string          `json:"content,omitempty"`
	AppliedTo *TargetSelector `json:"appliedTo,omitempty"`
}

type IncidentWebhookConfig struct {
	Name                      string `json:"name,omitempty"`
	DescriptionPath           string `json:"descriptionPath,omitempty"`
	ClusterPath               string `json:"clusterPath,omitempty"`
	K8SNamespacePath          string `json:"k8sNamespacePath,omitempty"`
	ArgocdApplicationNamePath string `json:"argocdApplicationNamePath,omitempty"`
}

type IncidentsConfig struct {
	Triggers []*TargetSelector        `json:"triggers,omitempty"`
	Webhooks []*IncidentWebhookConfig `json:"webhooks,omitempty"`
}

type AIConfig struct {
	Runbooks  []*Runbook       `json:"runbooks,omitempty"`
	Incidents *IncidentsConfig `json:"incidents,omitempty"`
}

type KubeVisionConfig struct {
	CveScanConfig *CveScanConfig `json:"cveScanConfig,omitempty"`
	AiConfig      *AIConfig      `json:"aiConfig,omitempty"`
}

type AppInAnyNamespaceConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type CustomDeprecatedAPI struct {
	ApiVersion                     string `json:"apiVersion,omitempty"`
	NewApiVersion                  string `json:"newApiVersion,omitempty"`
	DeprecatedInKubernetesVersion  string `json:"deprecatedInKubernetesVersion,omitempty"`
	UnavailableInKubernetesVersion string `json:"unavailableInKubernetesVersion,omitempty"`
}

type CveScanConfig struct {
	ScanEnabled    *bool  `json:"scanEnabled,omitempty"`
	RescanInterval string `json:"rescanInterval,omitempty"`
}

type ObjectSelector struct {
	MatchLabels      map[string]string           `json:"matchLabels,omitempty"`
	MatchExpressions []*LabelSelectorRequirement `json:"matchExpressions,omitempty"`
}

type LabelSelectorRequirement struct {
	Key      *string  `json:"key,omitempty"`
	Operator *string  `json:"operator,omitempty"`
	Values   []string `json:"values,omitempty"`
}

type ClusterSecretMapping struct {
	Clusters *ObjectSelector `json:"clusters,omitempty"`
	Secrets  *ObjectSelector `json:"secrets,omitempty"`
}

type SecretsManagementConfig struct {
	Sources      []*ClusterSecretMapping `json:"sources,omitempty"`
	Destinations []*ClusterSecretMapping `json:"destinations,omitempty"`
}

type ApplicationSetExtension struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type BucketRateLimiting struct {
	Enabled    *bool  `json:"enabled,omitempty"`
	BucketSize uint32 `json:"bucketSize,omitempty"`
	BucketQps  uint32 `json:"bucketQps,omitempty"`
}

type ItemRateLimiting struct {
	Enabled         *bool   `json:"enabled,omitempty"`
	FailureCooldown uint32  `json:"failureCooldown,omitempty"`
	BaseDelay       uint32  `json:"baseDelay,omitempty"`
	MaxDelay        uint32  `json:"maxDelay,omitempty"`
	BackoffFactor   float32 `json:"backoffFactor,omitempty"`
}

type AppReconciliationsRateLimiting struct {
	BucketRateLimiting *BucketRateLimiting `json:"bucketRateLimiting,omitempty"`
	ItemRateLimiting   *ItemRateLimiting   `json:"itemRateLimiting,omitempty"`
}

type InstanceSpec struct {
	IpAllowList                     []*IPAllowListEntry             `json:"ipAllowList,omitempty"`
	Subdomain                       string                          `json:"subdomain,omitempty"`
	DeclarativeManagementEnabled    *bool                           `json:"declarativeManagementEnabled,omitempty"`
	Extensions                      []*ArgoCDExtensionInstallEntry  `json:"extensions,omitempty"`
	ClusterCustomizationDefaults    *ClusterCustomization           `json:"clusterCustomizationDefaults,omitempty"`
	ImageUpdaterEnabled             *bool                           `json:"imageUpdaterEnabled,omitempty"`
	BackendIpAllowListEnabled       *bool                           `json:"backendIpAllowListEnabled,omitempty"`
	RepoServerDelegate              *RepoServerDelegate             `json:"repoServerDelegate,omitempty"`
	AuditExtensionEnabled           *bool                           `json:"auditExtensionEnabled,omitempty"`
	SyncHistoryExtensionEnabled     *bool                           `json:"syncHistoryExtensionEnabled,omitempty"`
	CrossplaneExtension             *CrossplaneExtension            `json:"crossplaneExtension,omitempty"`
	ImageUpdaterDelegate            *ImageUpdaterDelegate           `json:"imageUpdaterDelegate,omitempty"`
	AppSetDelegate                  *AppSetDelegate                 `json:"appSetDelegate,omitempty"`
	AssistantExtensionEnabled       *bool                           `json:"assistantExtensionEnabled,omitempty"`
	AppsetPolicy                    *AppsetPolicy                   `json:"appsetPolicy,omitempty"`
	HostAliases                     []*HostAliases                  `json:"hostAliases,omitempty"`
	AgentPermissionsRules           []*AgentPermissionsRule         `json:"agentPermissionsRules,omitempty"`
	Fqdn                            *string                         `json:"fqdn,omitempty"`
	MultiClusterK8SDashboardEnabled *bool                           `json:"multiClusterK8sDashboardEnabled,omitempty"`
	AkuityIntelligenceExtension     *AkuityIntelligenceExtension    `json:"akuityIntelligenceExtension,omitempty"`
	ImageUpdaterVersion             string                          `json:"imageUpdaterVersion,omitempty"`
	CustomDeprecatedApis            []*CustomDeprecatedAPI          `json:"customDeprecatedApis,omitempty"`
	KubeVisionConfig                *KubeVisionConfig               `json:"kubeVisionConfig,omitempty"`
	AppInAnyNamespaceConfig         *AppInAnyNamespaceConfig        `json:"appInAnyNamespaceConfig,omitempty"`
	Basepath                        string                          `json:"basepath,omitempty"`
	AppsetProgressiveSyncsEnabled   *bool                           `json:"appsetProgressiveSyncsEnabled,omitempty"`
	Secrets                         *SecretsManagementConfig        `json:"secrets,omitempty"`
	AppsetPlugins                   []*AppsetPlugins                `json:"appsetPlugins,omitempty"`
	ApplicationSetExtension         *ApplicationSetExtension        `json:"applicationSetExtension,omitempty"`
	AppReconciliationsRateLimiting  *AppReconciliationsRateLimiting `json:"appReconciliationsRateLimiting,omitempty"`
}

type AppsetPlugins struct {
	Name           string `json:"name,omitempty"`
	Token          string `json:"token,omitempty"`
	BaseUrl        string `json:"baseUrl,omitempty"`
	RequestTimeout int32  `json:"requestTimeout,omitempty"`
}

type ManagedCluster struct {
	ClusterName string `json:"clusterName,omitempty"`
}

type RepoServerDelegate struct {
	ControlPlane   *bool           `json:"controlPlane,omitempty"`
	ManagedCluster *ManagedCluster `json:"managedCluster,omitempty"`
}

type ImageUpdaterDelegate struct {
	ControlPlane   *bool           `json:"controlPlane,omitempty"`
	ManagedCluster *ManagedCluster `json:"managedCluster,omitempty"`
}

type AppSetDelegate struct {
	ManagedCluster *ManagedCluster `json:"managedCluster,omitempty"`
}

type IPAllowListEntry struct {
	Ip          string `json:"ip,omitempty"`
	Description string `json:"description,omitempty"`
}

type HostAliases struct {
	Ip        string   `json:"ip,omitempty"`
	Hostnames []string `json:"hostnames,omitempty"`
}

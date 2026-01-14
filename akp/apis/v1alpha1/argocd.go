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
	AutoUpgradeDisabled   *bool                `json:"autoUpgradeDisabled,omitempty"`
	Kustomization         runtime.RawExtension `json:"kustomization,omitempty"`
	AppReplication        *bool                `json:"appReplication,omitempty"`
	RedisTunneling        *bool                `json:"redisTunneling,omitempty"`
	ServerSideDiffEnabled *bool                `json:"serverSideDiffEnabled,omitempty"`
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
}

type ClusterAddonsExtension struct {
	Enabled          *bool    `json:"enabled,omitempty"`
	AllowedUsernames []string `json:"allowedUsernames,omitempty"`
	AllowedGroups    []string `json:"allowedGroups,omitempty"`
}

type TargetSelector struct {
	ArgocdApplications []string `json:"argocdApplications,omitempty"`
	K8SNamespaces      []string `json:"k8sNamespaces,omitempty"`
	Clusters           []string `json:"clusters,omitempty"`
	DegradedFor        *string  `json:"degradedFor,omitempty"`
}

type Runbook struct {
	Name              string          `json:"name,omitempty"`
	Content           string          `json:"content,omitempty"`
	AppliedTo         *TargetSelector `json:"appliedTo,omitempty"`
	SlackChannelNames []string        `json:"slackChannelNames,omitempty"`
}

type IncidentWebhookConfig struct {
	Name                      string `json:"name,omitempty"`
	DescriptionPath           string `json:"descriptionPath,omitempty"`
	ClusterPath               string `json:"clusterPath,omitempty"`
	K8SNamespacePath          string `json:"k8sNamespacePath,omitempty"`
	ArgocdApplicationNamePath string `json:"argocdApplicationNamePath,omitempty"`
}

type IncidentsGroupingConfig struct {
	K8SNamespaces          []string `json:"k8sNamespaces,omitempty"`
	ArgocdApplicationNames []string `json:"argocdApplicationNames,omitempty"`
}

type IncidentsConfig struct {
	Triggers []*TargetSelector        `json:"triggers,omitempty"`
	Webhooks []*IncidentWebhookConfig `json:"webhooks,omitempty"`
	Grouping *IncidentsGroupingConfig `json:"grouping,omitempty"`
}

type AIConfig struct {
	Runbooks            []*Runbook       `json:"runbooks,omitempty"`
	Incidents           *IncidentsConfig `json:"incidents,omitempty"`
	ArgocdSlackService  *string          `json:"argocdSlackService,omitempty"`
	ArgocdSlackChannels []string         `json:"argocdSlackChannels,omitempty"`
}

type AdditionalAttributeRule struct {
	Group       string   `json:"group,omitempty"`
	Kind        string   `json:"kind,omitempty"`
	Annotations []string `json:"annotations,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Namespace   string   `json:"namespace,omitempty"`
}

type KubeVisionConfig struct {
	CveScanConfig        *CveScanConfig             `json:"cveScanConfig,omitempty"`
	AiConfig             *AIConfig                  `json:"aiConfig,omitempty"`
	AdditionalAttributes []*AdditionalAttributeRule `json:"additionalAttributes,omitempty"`
}

type AppInAnyNamespaceConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type CveScanConfig struct {
	ScanEnabled    *bool  `json:"scanEnabled,omitempty"`
	RescanInterval string `json:"rescanInterval,omitempty"`
}

type ApplicationSetExtension struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type InstanceSpec struct {
	IpAllowList                     []*IPAllowListEntry            `json:"ipAllowList,omitempty"`
	Subdomain                       string                         `json:"subdomain,omitempty"`
	DeclarativeManagementEnabled    *bool                          `json:"declarativeManagementEnabled,omitempty"`
	Extensions                      []*ArgoCDExtensionInstallEntry `json:"extensions,omitempty"`
	ClusterCustomizationDefaults    *ClusterCustomization          `json:"clusterCustomizationDefaults,omitempty"`
	ImageUpdaterEnabled             *bool                          `json:"imageUpdaterEnabled,omitempty"`
	BackendIpAllowListEnabled       *bool                          `json:"backendIpAllowListEnabled,omitempty"`
	RepoServerDelegate              *RepoServerDelegate            `json:"repoServerDelegate,omitempty"`
	AuditExtensionEnabled           *bool                          `json:"auditExtensionEnabled,omitempty"`
	SyncHistoryExtensionEnabled     *bool                          `json:"syncHistoryExtensionEnabled,omitempty"`
	CrossplaneExtension             *CrossplaneExtension           `json:"crossplaneExtension,omitempty"`
	ImageUpdaterDelegate            *ImageUpdaterDelegate          `json:"imageUpdaterDelegate,omitempty"`
	AppSetDelegate                  *AppSetDelegate                `json:"appSetDelegate,omitempty"`
	AssistantExtensionEnabled       *bool                          `json:"assistantExtensionEnabled,omitempty"`
	AppsetPolicy                    *AppsetPolicy                  `json:"appsetPolicy,omitempty"`
	HostAliases                     []*HostAliases                 `json:"hostAliases,omitempty"`
	AgentPermissionsRules           []*AgentPermissionsRule        `json:"agentPermissionsRules,omitempty"`
	Fqdn                            *string                        `json:"fqdn,omitempty"`
	MultiClusterK8SDashboardEnabled *bool                          `json:"multiClusterK8sDashboardEnabled,omitempty"`
	AkuityIntelligenceExtension     *AkuityIntelligenceExtension   `json:"akuityIntelligenceExtension,omitempty"`

	KubeVisionConfig        *KubeVisionConfig        `json:"kubeVisionConfig,omitempty"`
	AppInAnyNamespaceConfig *AppInAnyNamespaceConfig `json:"appInAnyNamespaceConfig,omitempty"`

	AppsetPlugins           []*AppsetPlugins         `json:"appsetPlugins,omitempty"`
	ApplicationSetExtension *ApplicationSetExtension `json:"applicationSetExtension,omitempty"`

	MetricsIngressUsername        *string                 `json:"metricsIngressUsername,omitempty"`
	MetricsIngressPasswordHash    *string                 `json:"metricsIngressPasswordHash,omitempty"`
	PrivilegedNotificationCluster *string                 `json:"privilegedNotificationCluster,omitempty"`
	ClusterAddonsExtension        *ClusterAddonsExtension `json:"clusterAddonsExtension,omitempty"`
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

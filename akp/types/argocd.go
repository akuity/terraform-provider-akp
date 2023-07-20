// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2023 Akuity, Inc.
*/

package types

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ArgoCD is the Schema for the argocd API
type ArgoCD struct {
	Spec ArgoCDSpec `json:"spec" tfsdk:"spec"`
}

type ArgoCDSpec struct {
	Description  types.String `json:"description" tfsdk:"description"`
	Version      types.String `json:"version" tfsdk:"version"`
	InstanceSpec InstanceSpec `json:"instanceSpec,omitempty" tfsdk:"instance_spec"`
}

type ArgoCDExtensionInstallEntry struct {
	Id      types.String `json:"id,omitempty" tfsdk:"id"`
	Version types.String `json:"version,omitempty" tfsdk:"version"`
}

type ClusterCustomization struct {
	AutoUpgradeDisabled types.Bool   `json:"autoUpgradeDisabled,omitempty" tfsdk:"auto_upgrade_disabled"`
	Kustomization       types.String `json:"kustomization,omitempty" tfsdk:"kustomization"`
	AppReplication      types.Bool   `json:"appReplication,omitempty" tfsdk:"app_replication"`
	RedisTunneling      types.Bool   `json:"redisTunneling,omitempty" tfsdk:"redis_tunneling"`
}

type InstanceSpec struct {
	IpAllowList                  []*IPAllowListEntry            `json:"ipAllowList,omitempty" tfsdk:"ip_allow_list"`
	Subdomain                    types.String                   `json:"subdomain,omitempty" tfsdk:"subdomain"`
	DeclarativeManagementEnabled types.Bool                     `json:"declarativeManagementEnabled,omitempty" tfsdk:"declarative_management_enabled"`
	Extensions                   []*ArgoCDExtensionInstallEntry `json:"extensions,omitempty" tfsdk:"extensions"`
	ClusterCustomizationDefaults types.Object                   `json:"clusterCustomizationDefaults,omitempty" tfsdk:"cluster_customization_defaults"`
	ImageUpdaterEnabled          types.Bool                     `json:"imageUpdaterEnabled,omitempty" tfsdk:"image_updater_enabled"`
	BackendIpAllowListEnabled    types.Bool                     `json:"backendIpAllowListEnabled,omitempty" tfsdk:"backend_ip_allow_list_enabled"`
	RepoServerDelegate           *RepoServerDelegate            `json:"repoServerDelegate,omitempty" tfsdk:"repo_server_delegate"`
	AuditExtensionEnabled        types.Bool                     `json:"auditExtensionEnabled,omitempty" tfsdk:"audit_extension_enabled"`
	SyncHistoryExtensionEnabled  types.Bool                     `json:"syncHistoryExtensionEnabled,omitempty" tfsdk:"sync_history_extension_enabled"`
	ImageUpdaterDelegate         *ImageUpdaterDelegate          `json:"imageUpdaterDelegate,omitempty" tfsdk:"image_updater_delegate"`
	AppSetDelegate               *AppSetDelegate                `json:"appSetDelegate,omitempty" tfsdk:"app_set_delegate"`
}

type ManagedCluster struct {
	ClusterName types.String `json:"clusterName,omitempty" tfsdk:"cluster_name"`
}

type RepoServerDelegate struct {
	ControlPlane   types.Bool      `json:"controlPlane,omitempty" tfsdk:"control_plane"`
	ManagedCluster *ManagedCluster `json:"managedCluster,omitempty" tfsdk:"managed_cluster"`
}

type ImageUpdaterDelegate struct {
	ControlPlane   types.Bool      `json:"controlPlane,omitempty" tfsdk:"control_plane"`
	ManagedCluster *ManagedCluster `json:"managedCluster,omitempty" tfsdk:"managed_cluster"`
}

type AppSetDelegate struct {
	ManagedCluster *ManagedCluster `json:"managedCluster,omitempty" tfsdk:"managed_cluster"`
}

type IPAllowListEntry struct {
	Ip          types.String `json:"ip,omitempty" tfsdk:"ip"`
	Description types.String `json:"description,omitempty" tfsdk:"description"`
}

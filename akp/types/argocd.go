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
	Shard        types.String `tfsdk:"shard"`
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

type InstanceSpec struct {
	IpAllowList                  []*IPAllowListEntry            `tfsdk:"ip_allow_list"`
	Subdomain                    types.String                   `tfsdk:"subdomain"`
	DeclarativeManagementEnabled types.Bool                     `tfsdk:"declarative_management_enabled"`
	Extensions                   []*ArgoCDExtensionInstallEntry `tfsdk:"extensions"`
	ClusterCustomizationDefaults types.Object                   `tfsdk:"cluster_customization_defaults"`
	ImageUpdaterEnabled          types.Bool                     `tfsdk:"image_updater_enabled"`
	BackendIpAllowListEnabled    types.Bool                     `tfsdk:"backend_ip_allow_list_enabled"`
	RepoServerDelegate           *RepoServerDelegate            `tfsdk:"repo_server_delegate"`
	AuditExtensionEnabled        types.Bool                     `tfsdk:"audit_extension_enabled"`
	SyncHistoryExtensionEnabled  types.Bool                     `tfsdk:"sync_history_extension_enabled"`
	ImageUpdaterDelegate         *ImageUpdaterDelegate          `tfsdk:"image_updater_delegate"`
	AppSetDelegate               *AppSetDelegate                `tfsdk:"app_set_delegate"`
	AssistantExtensionEnabled    types.Bool                     `tfsdk:"assistant_extension_enabled"`
	AppsetPolicy                 *AppsetPolicy                  `tfsdk:"appset_policy"`
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

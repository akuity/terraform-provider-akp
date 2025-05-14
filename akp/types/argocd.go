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

type AppInAnyNamespaceConfig struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type InstanceSpec struct {
	IpAllowList                     []*IPAllowListEntry      `tfsdk:"ip_allow_list"`
	Subdomain                       types.String             `tfsdk:"subdomain"`
	DeclarativeManagementEnabled    types.Bool               `tfsdk:"declarative_management_enabled"`
	Extensions                      types.List               `tfsdk:"extensions"`
	ClusterCustomizationDefaults    types.Object             `tfsdk:"cluster_customization_defaults"`
	ImageUpdaterEnabled             types.Bool               `tfsdk:"image_updater_enabled"`
	BackendIpAllowListEnabled       types.Bool               `tfsdk:"backend_ip_allow_list_enabled"`
	RepoServerDelegate              *RepoServerDelegate      `tfsdk:"repo_server_delegate"`
	AuditExtensionEnabled           types.Bool               `tfsdk:"audit_extension_enabled"`
	SyncHistoryExtensionEnabled     types.Bool               `tfsdk:"sync_history_extension_enabled"`
	CrossplaneExtension             *CrossplaneExtension     `tfsdk:"crossplane_extension"`
	ImageUpdaterDelegate            *ImageUpdaterDelegate    `tfsdk:"image_updater_delegate"`
	AppSetDelegate                  *AppSetDelegate          `tfsdk:"app_set_delegate"`
	AssistantExtensionEnabled       types.Bool               `tfsdk:"assistant_extension_enabled"`
	AppsetPolicy                    types.Object             `tfsdk:"appset_policy"`
	HostAliases                     []*HostAliases           `tfsdk:"host_aliases"`
	AgentPermissionsRules           []*AgentPermissionsRule  `tfsdk:"agent_permissions_rules"`
	Fqdn                            types.String             `tfsdk:"fqdn"`
	MultiClusterK8SDashboardEnabled types.Bool               `tfsdk:"multi_cluster_k8s_dashboard_enabled"`
	AppInAnyNamespaceConfig         *AppInAnyNamespaceConfig `tfsdk:"app_in_any_namespace_config"`
	AppsetPlugins                   []*AppsetPlugins         `tfsdk:"appset_plugins"`
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

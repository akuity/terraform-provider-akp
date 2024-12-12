// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2023 Akuity, Inc.
*/

package types

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type Cluster struct {
	ID                            types.String `tfsdk:"id"`
	InstanceID                    types.String `tfsdk:"instance_id"`
	Name                          types.String `tfsdk:"name"`
	Namespace                     types.String `tfsdk:"namespace"`
	Labels                        types.Map    `tfsdk:"labels"`
	Annotations                   types.Map    `tfsdk:"annotations"`
	Spec                          *ClusterSpec `tfsdk:"spec"`
	Kubeconfig                    *Kubeconfig  `tfsdk:"kube_config"`
	RemoveAgentResourcesOnDestroy types.Bool   `tfsdk:"remove_agent_resources_on_destroy"`
}

type Clusters struct {
	ID         types.String `tfsdk:"id"`
	InstanceID types.String `tfsdk:"instance_id"`
	Clusters   []Cluster    `tfsdk:"clusters"`
}

type ClusterSpec struct {
	Description     types.String `tfsdk:"description"`
	NamespaceScoped types.Bool   `tfsdk:"namespace_scoped"`
	Data            ClusterData  `tfsdk:"data"`
}

type Resources struct {
	Mem types.String `tfsdk:"mem"`
	Cpu types.String `tfsdk:"cpu"`
}

type ManagedClusterConfig struct {
	SecretName types.String `tfsdk:"secret_name"`
	SecretKey  types.String `tfsdk:"secret_key"`
}

type AutoScalerConfig struct {
	ApplicationController *AppControllerAutoScalingConfig `tfsdk:"application_controller"`
	RepoServer            *RepoServerAutoScalingConfig    `tfsdk:"repo_server"`
}

type AppControllerAutoScalingConfig struct {
	ResourceMinimum *Resources `tfsdk:"resource_minimum"`
	ResourceMaximum *Resources `tfsdk:"resource_maximum"`
}

type RepoServerAutoScalingConfig struct {
	ResourceMinimum *Resources  `tfsdk:"resource_minimum"`
	ResourceMaximum *Resources  `tfsdk:"resource_maximum"`
	ReplicaMaximum  types.Int64 `tfsdk:"replica_maximum"`
	ReplicaMinimum  types.Int64 `tfsdk:"replica_minimum"`
}

type CustomAgentSizeConfig struct {
	ApplicationController *AppControllerCustomAgentSizeConfig `tfsdk:"application_controller"`
	RepoServer            *RepoServerCustomAgentSizeConfig    `tfsdk:"repo_server"`
}

type AppControllerCustomAgentSizeConfig struct {
	Mem types.String `tfsdk:"mem"`
	Cpu types.String `tfsdk:"cpu"`
}

type RepoServerCustomAgentSizeConfig struct {
	Mem     types.String `tfsdk:"mem"`
	Cpu     types.String `tfsdk:"cpu"`
	Replica types.Int64  `tfsdk:"replica"`
}

type ClusterData struct {
	Size                            types.String           `tfsdk:"size"`
	AutoUpgradeDisabled             types.Bool             `tfsdk:"auto_upgrade_disabled"`
	Kustomization                   types.String           `tfsdk:"kustomization"`
	AppReplication                  types.Bool             `tfsdk:"app_replication"`
	TargetVersion                   types.String           `tfsdk:"target_version"`
	RedisTunneling                  types.Bool             `tfsdk:"redis_tunneling"`
	DatadogAnnotationsEnabled       types.Bool             `tfsdk:"datadog_annotations_enabled"`
	EksAddonEnabled                 types.Bool             `tfsdk:"eks_addon_enabled"`
	ManagedClusterConfig            *ManagedClusterConfig  `tfsdk:"managed_cluster_config"`
	MultiClusterK8SDashboardEnabled types.Bool             `tfsdk:"multi_cluster_k8s_dashboard_enabled"`
	AutoscalerConfig                basetypes.ObjectValue  `tfsdk:"auto_agent_size_config"`
	CustomAgentSizeConfig           *CustomAgentSizeConfig `tfsdk:"custom_agent_size_config"`
}

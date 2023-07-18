// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2023 Akuity, Inc.
*/

package types

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Cluster is the Schema for the cluster API
type Cluster struct {
	Name        types.String `json:"name,omitempty" tfsdk:"name"`
	Namespace   types.String `json:"namespace,omitempty" tfsdk:"namespace"`
	Labels      types.Map    `json:"labels,omitempty" tfsdk:"labels"`
	Annotations types.Map    `json:"annotations,omitempty" tfsdk:"annotations"`
	Spec        ClusterSpec  `json:"spec" tfsdk:"spec"`
	Kubeconfig  types.Object `json:"kubeconfig,omitempty" tfsdk:"kubeconfig"`
	Manifests   types.String `json:"manifests,omitempty" tfsdk:"manifests"`
}

type ClusterSpec struct {
	Description     types.String `json:"description,omitempty" tfsdk:"description"`
	NamespaceScoped types.Bool   `json:"namespaceScoped,omitempty" tfsdk:"namespace_scoped"`
	Data            ClusterData  `json:"data,omitempty" tfsdk:"data"`
}

type ClusterData struct {
	Size                types.String `json:"size,omitempty" tfsdk:"size"`
	AutoUpgradeDisabled types.Bool   `json:"autoUpgradeDisabled,omitempty" tfsdk:"auto_upgrade_disabled"`
	Kustomization       types.String `json:"kustomization,omitempty" tfsdk:"kustomization"`
	AppReplication      types.Bool   `json:"appReplication,omitempty" tfsdk:"app_replication"`
	TargetVersion       types.String `json:"targetVersion,omitempty" tfsdk:"target_version"`
	RedisTunneling      types.Bool   `json:"redisTunneling,omitempty" tfsdk:"redis_tunneling"`
}

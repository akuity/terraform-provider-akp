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
	ID          types.String `json:"id,omitempty" tfsdk:"id"`
	InstanceID  types.String `tfsdk:"instance_id"`
	Name        types.String `tfsdk:"name"`
	Namespace   types.String `tfsdk:"namespace"`
	Labels      types.Map    `tfsdk:"labels"`
	Annotations types.Map    `tfsdk:"annotations"`
	Spec        *ClusterSpec `tfsdk:"spec"`
	Kubeconfig  *Kubeconfig  `tfsdk:"kubeconfig"`
	Manifests   types.String `tfsdk:"manifests"`
}

type Clusters struct {
	InstanceID types.String `tfsdk:"instance_id"`
	Clusters   []Cluster    `tfsdk:"clusters"`
}

type ClusterSpec struct {
	Description     types.String `tfsdk:"description"`
	NamespaceScoped types.Bool   `tfsdk:"namespace_scoped"`
	Data            ClusterData  `tfsdk:"data"`
}

type ClusterData struct {
	Size                types.String `tfsdk:"size"`
	AutoUpgradeDisabled types.Bool   `tfsdk:"auto_upgrade_disabled"`
	Kustomization       types.String `tfsdk:"kustomization"`
	AppReplication      types.Bool   `tfsdk:"app_replication"`
	TargetVersion       types.String `tfsdk:"target_version"`
	RedisTunneling      types.Bool   `tfsdk:"redis_tunneling"`
}

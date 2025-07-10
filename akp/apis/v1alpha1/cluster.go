// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2023 Akuity, Inc.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterSpec `json:"spec,omitempty"`
}

type ClusterSize string

type ClusterSpec struct {
	Description     string      `json:"description,omitempty"`
	NamespaceScoped bool        `json:"namespaceScoped,omitempty"`
	Data            ClusterData `json:"data,omitempty"`
}

type Resources struct {
	Mem string `json:"mem,omitempty"`
	Cpu string `json:"cpu,omitempty"`
}

type ManagedClusterConfig struct {
	SecretName string `json:"secretName,omitempty"`
	SecretKey  string `json:"secretKey,omitempty"`
}

type AutoScalerConfig struct {
	ApplicationController *AppControllerAutoScalingConfig `json:"applicationController,omitempty"`
	RepoServer            *RepoServerAutoScalingConfig    `json:"repoServer,omitempty"`
}

type AppControllerAutoScalingConfig struct {
	ResourceMinimum *Resources `json:"resourceMinimum,omitempty"`
	ResourceMaximum *Resources `json:"resourceMaximum,omitempty"`
}

type RepoServerAutoScalingConfig struct {
	ResourceMinimum *Resources `json:"resourceMinimum,omitempty"`
	ResourceMaximum *Resources `json:"resourceMaximum,omitempty"`
	ReplicaMaximum  int32      `json:"replicaMaximum,omitempty"`
	ReplicaMinimum  int32      `json:"replicaMinimum,omitempty"`
}

type ClusterCompatibility struct {
	Ipv6Only bool `json:"ipv6Only,omitempty"`
}

type ClusterArgoCDNotificationsSettings struct {
	InClusterSettings bool `json:"inClusterSettings,omitempty"`
}

type ClusterData struct {
	Size                ClusterSize          `json:"size,omitempty"`
	AutoUpgradeDisabled *bool                `json:"autoUpgradeDisabled,omitempty"`
	Kustomization       runtime.RawExtension `json:"kustomization,omitempty"`
	AppReplication      *bool                `json:"appReplication,omitempty"`
	TargetVersion       string               `json:"targetVersion,omitempty"`
	RedisTunneling      *bool                `json:"redisTunneling,omitempty"`

	DatadogAnnotationsEnabled *bool                 `json:"datadogAnnotationsEnabled,omitempty"`
	EksAddonEnabled           *bool                 `json:"eksAddonEnabled,omitempty"`
	ManagedClusterConfig      *ManagedClusterConfig `json:"managedClusterConfig,omitempty"`

	MultiClusterK8SDashboardEnabled *bool                               `json:"multiClusterK8sDashboardEnabled,omitempty"`
	AutoscalerConfig                *AutoScalerConfig                   `json:"autoscalerConfig,omitempty"`
	Project                         string                              `json:"project,omitempty"`
	Compatibility                   *ClusterCompatibility               `json:"compatibility,omitempty"`
	ArgocdNotificationsSettings     *ClusterArgoCDNotificationsSettings `json:"argocdNotificationsSettings,omitempty"`
}

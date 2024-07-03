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

type ManagedClusterConfig struct {
	SecretName string `json:"secretName,omitempty"`
	SecretKey  string `json:"secretKey,omitempty"`
}

type ClusterData struct {
	Size                      ClusterSize           `json:"size,omitempty"`
	AutoUpgradeDisabled       *bool                 `json:"autoUpgradeDisabled,omitempty"`
	Kustomization             runtime.RawExtension  `json:"kustomization,omitempty"`
	AppReplication            *bool                 `json:"appReplication,omitempty"`
	TargetVersion             string                `json:"targetVersion,omitempty"`
	RedisTunneling            *bool                 `json:"redisTunneling,omitempty"`
	DatadogAnnotationsEnabled *bool                 `json:"datadogAnnotationsEnabled,omitempty"`
	EksAddonEnabled           *bool                 `json:"eksAddonEnabled,omitempty"`
	ManagedClusterConfig      *ManagedClusterConfig `json:"managedClusterConfig,omitempty"`
}

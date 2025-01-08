// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2023 Akuity, Inc.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Kargo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KargoSpec `json:"spec,omitempty"`
}

type KargoSpec struct {
	Description       string            `json:"description"`
	Version           string            `json:"version"`
	KargoInstanceSpec KargoInstanceSpec `json:"kargoInstanceSpec,omitempty"`
}

type KargoIPAllowListEntry struct {
	Ip          string `json:"ip,omitempty"`
	Description string `json:"description,omitempty"`
}

type KargoAgentCustomization struct {
	AutoUpgradeDisabled *bool                `json:"autoUpgradeDisabled,omitempty"`
	Kustomization       runtime.RawExtension `json:"kustomization,omitempty"`
}

type KargoInstanceSpec struct {
	BackendIpAllowListEnabled  *bool                    `json:"backendIpAllowListEnabled,omitempty"`
	IpAllowList                []*KargoIPAllowListEntry `json:"ipAllowList,omitempty"`
	AgentCustomizationDefaults *KargoAgentCustomization `json:"agentCustomizationDefaults,omitempty"`
	DefaultShardAgent          string                   `json:"defaultShardAgent,omitempty"`
	GlobalCredentialsNs        []string                 `json:"globalCredentialsNs,omitempty"`
	GlobalServiceAccountNs     []string                 `json:"globalServiceAccountNs,omitempty"`
}

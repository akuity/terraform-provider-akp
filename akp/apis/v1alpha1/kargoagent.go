// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2023 Akuity, Inc.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type KargoAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KargoAgentSpec `json:"spec,omitempty"`
}

type KargoAgentSize string

type KargoAgentSpec struct {
	Description string         `json:"description,omitempty"`
	Data        KargoAgentData `json:"data,omitempty"`
}

type KargoAgentData struct {
	Size                KargoAgentSize       `json:"size,omitempty"`
	AutoUpgradeDisabled *bool                `json:"autoUpgradeDisabled,omitempty"`
	TargetVersion       string               `json:"targetVersion,omitempty"`
	Kustomization       runtime.RawExtension `json:"kustomization,omitempty"`
	RemoteArgocd        string               `json:"remoteArgocd,omitempty"`
	AkuityManaged       *bool                 `json:"akuityManaged,omitempty"`
	ArgocdNamespace     string               `json:"argocdNamespace,omitempty"`
}

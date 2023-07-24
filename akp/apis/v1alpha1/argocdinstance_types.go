// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2023 Akuity, Inc.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ArgoCD is the Schema for the argocd API
type ArgoCD struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ArgoCDSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// ArgoCDList contains a list of ArgoCD
type ArgoCDList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArgoCD `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArgoCD{}, &ArgoCDList{})
}

type ArgoCDSpec struct {
	Description  string       `json:"description"`
	Version      string       `json:"version"`
	InstanceSpec InstanceSpec `json:"instanceSpec,omitempty"`
}

type ArgoCDExtensionInstallEntry struct {
	Id      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
}

type ClusterCustomization struct {
	AutoUpgradeDisabled *bool                `json:"autoUpgradeDisabled,omitempty"`
	Kustomization       runtime.RawExtension `json:"kustomization,omitempty"`
	AppReplication      *bool                `json:"appReplication,omitempty"`
	RedisTunneling      *bool                `json:"redisTunneling,omitempty"`
}

type InstanceSpec struct {
	IpAllowList                  []*IPAllowListEntry            `json:"ipAllowList,omitempty"`
	Subdomain                    string                         `json:"subdomain,omitempty"`
	DeclarativeManagementEnabled *bool                          `json:"declarativeManagementEnabled,omitempty"`
	Extensions                   []*ArgoCDExtensionInstallEntry `json:"extensions,omitempty"`
	ClusterCustomizationDefaults *ClusterCustomization          `json:"clusterCustomizationDefaults,omitempty"`
	ImageUpdaterEnabled          *bool                          `json:"imageUpdaterEnabled,omitempty"`
	BackendIpAllowListEnabled    *bool                          `json:"backendIpAllowListEnabled,omitempty"`
	RepoServerDelegate           *RepoServerDelegate            `json:"repoServerDelegate,omitempty"`
	AuditExtensionEnabled        *bool                          `json:"auditExtensionEnabled,omitempty"`
	SyncHistoryExtensionEnabled  *bool                          `json:"syncHistoryExtensionEnabled,omitempty"`
	ImageUpdaterDelegate         *ImageUpdaterDelegate          `json:"imageUpdaterDelegate,omitempty"`
	AppSetDelegate               *AppSetDelegate                `json:"appSetDelegate,omitempty"`
}

type ManagedCluster struct {
	ClusterName string `json:"clusterName,omitempty"`
}

type RepoServerDelegate struct {
	ControlPlane   *bool           `json:"controlPlane,omitempty"`
	ManagedCluster *ManagedCluster `json:"managedCluster,omitempty"`
}

type ImageUpdaterDelegate struct {
	ControlPlane   *bool           `json:"controlPlane,omitempty"`
	ManagedCluster *ManagedCluster `json:"managedCluster,omitempty"`
}

type AppSetDelegate struct {
	ManagedCluster *ManagedCluster `json:"managedCluster,omitempty"`
}

type IPAllowListEntry struct {
	Ip          string `json:"ip,omitempty"`
	Description string `json:"description,omitempty"`
}
// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2025 Akuity, Inc.
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
	Fqdn              string            `json:"fqdn,omitempty"`
	Subdomain         string            `json:"subdomain,omitempty"`
	OidcConfig        *KargoOidcConfig  `json:"oidcConfig,omitempty"`
}

type KargoPredefinedAccountClaimValue struct {
	Values []string `json:"values"`
}

type KargoPredefinedAccountData struct {
	Claims map[string]KargoPredefinedAccountClaimValue `json:"claims,omitempty"`
}

type KargoOidcConfig struct {
	Enabled          *bool                      `json:"enabled"`
	DexEnabled       *bool                      `json:"dexEnabled"`
	DexConfig        string                     `json:"dexConfig"`
	DexConfigSecret  map[string]Value           `json:"dexConfigSecret"`
	IssuerURL        string                     `json:"issuerUrl"`
	ClientID         string                     `json:"clientId"`
	CliClientID      string                     `json:"cliClientId"`
	AdminAccount     KargoPredefinedAccountData `json:"adminAccount"`
	ViewerAccount    KargoPredefinedAccountData `json:"viewerAccount"`
	AdditionalScopes []string                   `json:"additionalScopes"`
}

type Value struct {
	Value *string `json:"value,omitempty"`
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

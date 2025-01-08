// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2025 Akuity, Inc.
*/

package types

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Kargo struct {
	Spec KargoSpec `tfsdk:"spec"`
}

type KargoSpec struct {
	Description       types.String      `tfsdk:"description"`
	Version           types.String      `tfsdk:"version"`
	KargoInstanceSpec KargoInstanceSpec `tfsdk:"kargo_instance_spec"`
}

type KargoIPAllowListEntry struct {
	Ip          types.String `tfsdk:"ip"`
	Description types.String `tfsdk:"description"`
}

type KargoAgentCustomization struct {
	AutoUpgradeDisabled types.Bool   `tfsdk:"auto_upgrade_disabled"`
	Kustomization       types.String `tfsdk:"kustomization"`
}

type KargoInstanceSpec struct {
	BackendIpAllowListEnabled  types.Bool               `tfsdk:"backend_ip_allow_list_enabled"`
	IpAllowList                []*KargoIPAllowListEntry `tfsdk:"ip_allow_list"`
	AgentCustomizationDefaults *KargoAgentCustomization `tfsdk:"agent_customization_defaults"`
	DefaultShardAgent          types.String             `tfsdk:"default_shard_agent"`
	GlobalCredentialsNs        []types.String           `tfsdk:"global_credentials_ns"`
	GlobalServiceAccountNs     []types.String           `tfsdk:"global_service_account_ns"`
}

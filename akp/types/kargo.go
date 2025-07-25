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
	Fqdn              types.String      `tfsdk:"fqdn"`
	Subdomain         types.String      `tfsdk:"subdomain"`
	OidcConfig        *KargoOidcConfig  `tfsdk:"oidc_config"`
}

type KargoPredefinedAccountClaimValue struct {
	Values []types.String `tfsdk:"values"`
}

type KargoPredefinedAccountData struct {
	Claims types.Map `tfsdk:"claims" schemaGen:"KargoPredefinedAccountClaimValue"`
}

type KargoOidcConfig struct {
	Enabled          types.Bool     `tfsdk:"enabled"`
	DexEnabled       types.Bool     `tfsdk:"dex_enabled"`
	DexConfig        types.String   `tfsdk:"dex_config"`
	DexConfigSecret  types.Map      `tfsdk:"dex_config_secret" schemaGen:"Value"`
	IssuerURL        types.String   `tfsdk:"issuer_url"`
	ClientID         types.String   `tfsdk:"client_id"`
	CliClientID      types.String   `tfsdk:"cli_client_id"`
	AdminAccount     types.Object   `tfsdk:"admin_account" schemaGen:"KargoPredefinedAccountData"`
	ViewerAccount    types.Object   `tfsdk:"viewer_account" schemaGen:"KargoPredefinedAccountData"`
	AdditionalScopes []types.String `tfsdk:"additional_scopes"`
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

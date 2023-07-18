package types

import "github.com/hashicorp/terraform-plugin-framework/types"

type Kubeconfig struct {
	Host                  types.String `tfsdk:"host"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	Insecure              types.Bool   `tfsdk:"insecure"`
	ClientCertificate     types.String `tfsdk:"client_certificate"`
	ClientKey             types.String `tfsdk:"client_key"`
	ClusterCaCertificate  types.String `tfsdk:"cluster_ca_certificate"`
	ConfigPath            types.String `tfsdk:"config_path"`
	ConfigPaths           types.List   `tfsdk:"config_paths"`
	ConfigContext         types.String `tfsdk:"config_context"`
	ConfigContextAuthInfo types.String `tfsdk:"config_context_auth_info"`
	ConfigContextCluster  types.String `tfsdk:"config_context_cluster"`
	Token                 types.String `tfsdk:"token"`
	ProxyUrl              types.String `tfsdk:"proxy_url"`
}

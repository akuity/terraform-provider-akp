package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (k *AkpKargoDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets information about a Kargo instance",
		Attributes:          getAKPKargoDataSourceAttributes(),
	}
}

func getAKPKargoDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "Kargo instance ID",
			Computed:            true,
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "Kargo instance name",
			Required:            true,
		},
		"kargo": schema.SingleNestedAttribute{
			MarkdownDescription: "Specification of the Kargo instance",
			Computed:            true,
			Attributes:          getKargoDataSourceAttributes(),
		},
		"kargo_cm": schema.MapAttribute{
			MarkdownDescription: "ConfigMap to configure system account accesses. The usage can be found in the examples/resources/akp_kargo_instance/resource.tf",
			ElementType:         types.StringType,
			Computed:            true,
		},
		"kargo_secret": schema.MapAttribute{
			MarkdownDescription: "Secret to configure system account accesses. The usage can be found in the examples/resources/akp_kargo_instance/resource.tf",
			ElementType:         types.StringType,
			Computed:            true,
		},
		"workspace": schema.StringAttribute{
			MarkdownDescription: "Workspace name for the Kargo instance",
			Computed:            true,
		},
		"kargo_resources": schema.MapAttribute{
			MarkdownDescription: "Map of Kargo custom resources to be managed alongside the Kargo instance. Currently supported resources are: `Project`, `ClusterPromotionTask`, `Stage`, `Warehouse`, `AnalysisTemplate`, `PromotionTask`. Should all be in the apiVersion `kargo.akuity.io/v1alpha1`.",
			Computed:            true,
			ElementType:         types.StringType,
		},
	}
}

func getKargoDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Kargo instance spec",
			Computed:            true,
			Attributes:          getKargoSpecDataSourceAttributes(),
		},
	}
}

func getKargoSpecDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"description": schema.StringAttribute{
			MarkdownDescription: "Description of the Kargo instance",
			Computed:            true,
		},
		"version": schema.StringAttribute{
			MarkdownDescription: "Version of the Kargo instance",
			Computed:            true,
		},
		"kargo_instance_spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Kargo instance specific configuration",
			Computed:            true,
			Attributes:          getKargoInstanceSpecDataSourceAttributes(),
		},
		"fqdn": schema.StringAttribute{
			MarkdownDescription: "FQDN of the Kargo instance",
			Computed:            true,
		},
		"subdomain": schema.StringAttribute{
			MarkdownDescription: "Subdomain of the Kargo instance",
			Computed:            true,
		},
		"oidc_config": schema.SingleNestedAttribute{
			MarkdownDescription: "OIDC configuration",
			Computed:            true,
			Attributes:          getOIDCConfigDataSourceAttributes(),
		},
	}
}

func getKargoInstanceSpecDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"backend_ip_allow_list_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether IP allow list is enabled for the backend",
			Computed:            true,
		},
		"ip_allow_list": schema.ListNestedAttribute{
			MarkdownDescription: "List of allowed IPs",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getKargoIPAllowListEntryDataSourceAttributes(),
			},
		},
		"agent_customization_defaults": schema.SingleNestedAttribute{
			MarkdownDescription: "Default agent customization settings",
			Computed:            true,
			Attributes:          getKargoAgentCustomizationDataSourceAttributes(),
		},
		"default_shard_agent": schema.StringAttribute{
			MarkdownDescription: "Default shard agent",
			Computed:            true,
		},
		"global_credentials_ns": schema.ListAttribute{
			MarkdownDescription: "List of global credentials namespaces",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"global_service_account_ns": schema.ListAttribute{
			MarkdownDescription: "List of global service account namespaces",
			Computed:            true,
			ElementType:         types.StringType,
		},
	}
}

func getKargoIPAllowListEntryDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ip": schema.StringAttribute{
			MarkdownDescription: "IP address",
			Computed:            true,
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "Description for the IP address",
			Computed:            true,
		},
	}
}

func getKargoAgentCustomizationDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"auto_upgrade_disabled": schema.BoolAttribute{
			MarkdownDescription: "Whether auto upgrade is disabled",
			Computed:            true,
		},
		"kustomization": schema.StringAttribute{
			MarkdownDescription: "Kustomization configuration",
			Computed:            true,
		},
	}
}

func getOIDCConfigDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether OIDC is enabled",
			Computed:            true,
		},
		"dex_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether DEX is enabled",
			Computed:            true,
		},
		"dex_config": schema.StringAttribute{
			MarkdownDescription: "DEX configuration",
			Computed:            true,
		},
		"dex_config_secret": schema.MapAttribute{
			MarkdownDescription: "DEX configuration secret",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"issuer_url": schema.StringAttribute{
			MarkdownDescription: "Issuer URL",
			Computed:            true,
		},
		"client_id": schema.StringAttribute{
			MarkdownDescription: "Client ID",
			Computed:            true,
		},
		"cli_client_id": schema.StringAttribute{
			MarkdownDescription: "CLI Client ID",
			Computed:            true,
		},
		"admin_account": schema.SingleNestedAttribute{
			MarkdownDescription: "Admin account",
			Computed:            true,
			Attributes:          getKargoPredefinedAccountDataAttributes(),
		},
		"viewer_account": schema.SingleNestedAttribute{
			MarkdownDescription: "Viewer account",
			Computed:            true,
			Attributes:          getKargoPredefinedAccountDataAttributes(),
		},
		"additional_scopes": schema.ListAttribute{
			MarkdownDescription: "Additional scopes",
			Computed:            true,
			ElementType:         types.StringType,
		},
	}
}

func getKargoPredefinedAccountDataAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"claims": schema.MapNestedAttribute{
			MarkdownDescription: "Claims",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"values": schema.ListAttribute{
						ElementType: types.StringType,
						Computed:    true,
					},
				},
			},
		},
	}
}

package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *AkpInstanceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Find an Argo CD instance by its name",
		Attributes:          getAKPInstanceDataSourceAttributes(),
	}
}

func getAKPInstanceDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Name",
		},
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Instance ID",
		},
		"argocd": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getArgoCDDataSourceAttributes(),
		},
		"argocd_cm": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getConfigMapDataSourceAttributes(),
		},
		"argocd_rbac_cm": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getConfigMapDataSourceAttributes(),
		},
		"argocd_secret": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getSecretDataSourceAttributes(),
		},
		"argocd_notifications_cm": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getConfigMapDataSourceAttributes(),
		},
		"argocd_notifications_secret": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getSecretDataSourceAttributes(),
		},
		"argocd_image_updater_config": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getConfigMapDataSourceAttributes(),
		},
		"argocd_image_updater_ssh_config": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getConfigMapDataSourceAttributes(),
		},
		"argocd_image_updater_secret": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getSecretDataSourceAttributes(),
		},
		"argocd_ssh_known_hosts_cm": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getConfigMapDataSourceAttributes(),
		},
		"argocd_tls_certs_cm": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getConfigMapDataSourceAttributes(),
		},
		"repo_credential_secrets": schema.ListNestedAttribute{
			Computed: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getSecretDataSourceAttributes(),
			},
		},
		"repo_template_credential_secrets": schema.ListNestedAttribute{
			Computed: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getSecretDataSourceAttributes(),
			},
		},
	}
}

func getConfigMapDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"data": schema.MapAttribute{
			ElementType: types.StringType,
			Computed:    true,
		},
	}
}

func getSecretDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Name",
		},
		"labels": schema.MapAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "Labels",
			Computed:            true,
		},
		"data": schema.MapAttribute{
			ElementType: types.StringType,
			Computed:    true,
			Sensitive:   true,
		},
		"string_data": schema.MapAttribute{
			ElementType: types.StringType,
			Computed:    true,
			Sensitive:   true,
		},
		"type": schema.StringAttribute{
			Computed: true,
		},
	}
}

func getArgoCDDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"spec": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getArgoCDSpecDataSourceAttributes(),
		},
	}
}

func getArgoCDSpecDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"description": schema.StringAttribute{
			Computed: true,
		},
		"version": schema.StringAttribute{
			Computed: true,
		},
		"instance_spec": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getInstanceSpecDataSourceAttributes(),
		},
	}
}

func getInstanceSpecDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ip_allow_list": schema.ListNestedAttribute{
			MarkdownDescription: "IP Allow List",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getIPAllowListEntryDataSourceAttributes(),
			},
		},
		"subdomain": schema.StringAttribute{
			MarkdownDescription: "Instance Subdomain. By default equals to instance id",
			Computed:            true,
		},
		"declarative_management_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Declarative Management",
			Computed:            true,
		},
		"extensions": schema.ListNestedAttribute{
			MarkdownDescription: "Extensions",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getArgoCDExtensionInstallEntryDataSourceAttributes(),
			},
		},
		"cluster_customization_defaults": schema.SingleNestedAttribute{
			MarkdownDescription: "Default Values For Cluster Agents",
			Computed:            true,
			Attributes:          getClusterCustomizationDataSourceAttributes(),
		},
		"image_updater_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Image Updater",
			Computed:            true,
		},
		"backend_ip_allow_list_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable IP Allow List to Cluster Agents",
			Computed:            true,
		},
		"repo_server_delegate": schema.SingleNestedAttribute{
			MarkdownDescription: "In case some clusters don't have network access to your private Git provider you can delegate these operations to one specific cluster.",
			Computed:            true,
			Attributes:          getRepoServerDelegateDataSourceAttributes(),
		},
		"audit_extension_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Audit Extension. Set this to `true` to install Audit Extension to Argo CD instance. Do not use `spec.extensions` for that",
			Computed:            true,
		},
		"sync_history_extension_enabled": schema.BoolAttribute{
			Computed: true,
		},
		"image_updater_delegate": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getImageUpdaterDelegateDataSourceAttributes(),
		},
		"app_set_delegate": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getAppSetDelegateDataSourceAttributes(),
		},
	}
}

func getIPAllowListEntryDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ip": schema.StringAttribute{
			MarkdownDescription: "IP Address",
			Computed:            true,
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "IP Description",
			Computed:            true,
		},
	}
}

func getArgoCDExtensionInstallEntryDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "Extension ID",
			Computed:            true,
		},
		"version": schema.StringAttribute{
			MarkdownDescription: "Extension version",
			Computed:            true,
		},
	}
}

func getClusterCustomizationDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"auto_upgrade_disabled": schema.BoolAttribute{
			MarkdownDescription: "Disable Agent Auto-upgrade",
			Computed:            true,
		},
		"kustomization": schema.StringAttribute{
			Computed: true,
		},
		"app_replication": schema.BoolAttribute{
			Computed: true,
		},
		"redis_tunneling": schema.BoolAttribute{
			Computed: true,
		},
	}
}

func getRepoServerDelegateDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"control_plane": schema.BoolAttribute{
			MarkdownDescription: "Use Control Plane",
			Computed:            true,
		},
		"managed_cluster": schema.SingleNestedAttribute{
			MarkdownDescription: "Use Managed Cluster",
			Computed:            true,
			Attributes:          getManagedClusterDataSourceAttributes(),
		},
	}
}

func getImageUpdaterDelegateDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"control_plane": schema.BoolAttribute{
			MarkdownDescription: "Use Control Plane",
			Computed:            true,
		},
		"managed_cluster": schema.SingleNestedAttribute{
			MarkdownDescription: "Use Managed Cluster",
			Computed:            true,
			Attributes:          getManagedClusterDataSourceAttributes(),
		},
	}
}

func getAppSetDelegateDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"managed_cluster": schema.SingleNestedAttribute{
			MarkdownDescription: "Use Managed Cluster",
			Computed:            true,
			Attributes:          getManagedClusterDataSourceAttributes(),
		},
	}
}

func getManagedClusterDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"cluster_name": schema.StringAttribute{
			MarkdownDescription: "Cluster Name",
			Computed:            true,
		},
	}
}

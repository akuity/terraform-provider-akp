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
		"id": schema.StringAttribute{
			Required:            true,
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
		"clusters": schema.ListNestedAttribute{
			Computed:  true,
			Sensitive: false,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getClusterDataSourceAttributes(),
			},
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
		"metadata": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Metadata of ConfigMap",
			Attributes:          getObjectMetaDataSourceAttributes(),
		},
		"data": schema.MapAttribute{
			ElementType: types.StringType,
			Computed:    true,
		},
	}
}

func getSecretDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"metadata": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Metadata of Secret",
			Attributes:          getSecretObjectMetaDataSourceAttributes(),
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
		"metadata": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "ArgoCD Instance Metadata",
			Attributes:          getObjectMetaDataSourceAttributes(),
		},
		"spec": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getArgoCDSpecDataSourceAttributes(),
		},
	}
}

func getClusterDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"metadata": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Cluster Metadata",
			Attributes:          getClusterObjectMetaDataSourceAttributes(),
		},
		"spec": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getClusterSpecDataSourceAttributes(),
		},
		"kubeconfig": schema.SingleNestedAttribute{
			MarkdownDescription: "Kubernetes connection settings. If configured, terraform will try to connect to the cluster and install the agent",
			Computed:            true,
			Attributes:          getKubeconfigDataSourceAttributes(),
		},
		"manifests": schema.StringAttribute{
			MarkdownDescription: "Agent Installation Manifests",
			Computed:            true,
			Sensitive:           true,
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

func getClusterSpecDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"description": schema.StringAttribute{
			MarkdownDescription: "Cluster Description",
			Computed:            true,
		},
		"namespace_scoped": schema.BoolAttribute{
			MarkdownDescription: "Agent Namespace Scoped",
			Computed:            true,
		},
		"data": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getClusterDataDataSourceAttributes(),
		},
	}
}

func getClusterDataDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"size": schema.StringAttribute{
			MarkdownDescription: "Cluster Size. One of `small`, `medium` or `large`",
			Computed:            true,
		},
		"auto_upgrade_disabled": schema.BoolAttribute{
			MarkdownDescription: "Disable Agents Auto Upgrade. On resource update terraform will try to update the agent if this is set to `true`. Otherwise agent will update itself automatically",
			Computed:            true,
		},
		"kustomization": schema.StringAttribute{
			Computed: true,
		},
		"app_replication": schema.BoolAttribute{
			Computed: true,
		},
		"target_version": schema.StringAttribute{
			MarkdownDescription: "Installed agent version",
			Computed:            true,
		},
		"redis_tunneling": schema.BoolAttribute{
			Computed: true,
		},
	}
}

func getClusterObjectMetaDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Name",
		},
		"namespace": schema.StringAttribute{
			MarkdownDescription: "Agent Installation Namespace",
			Computed:            true,
		},
		"labels": schema.MapAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "Labels",
			Computed:            true,
		},
		"annotations": schema.MapAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "Annotations",
			Computed:            true,
		},
	}
}

func getObjectMetaDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Name",
		},
		//"labels": schema.MapAttribute{
		//	ElementType:         types.StringType,
		//	MarkdownDescription: "Labels",
		//	Computed:            true,
		//	Computed:            true,
		//	PlanModifiers: []planmodifier.Map{
		//		mapplanmodifier.UseStateForUnknown(),
		//	},
		//},
	}
}

func getSecretObjectMetaDataSourceAttributes() map[string]schema.Attribute {
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
	}
}

func getKubeconfigDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"host": schema.StringAttribute{
			Computed:    true,
			Description: "The hostname (in form of URI) of Kubernetes master.",
		},
		"username": schema.StringAttribute{
			Computed:    true,
			Description: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
		},
		"password": schema.StringAttribute{
			Computed:    true,
			Sensitive:   true,
			Description: "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
		},
		"insecure": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether server should be accessed without verifying the TLS certificate.",
		},
		"client_certificate": schema.StringAttribute{
			Computed:    true,
			Description: "PEM-encoded client certificate for TLS authentication.",
		},
		"client_key": schema.StringAttribute{
			Computed:    true,
			Sensitive:   true,
			Description: "PEM-encoded client certificate key for TLS authentication.",
		},
		"cluster_ca_certificate": schema.StringAttribute{
			Computed:    true,
			Description: "PEM-encoded root certificates bundle for TLS authentication.",
		},
		"config_paths": schema.ListAttribute{
			ElementType: types.StringType,
			Computed:    true,
			Description: "A list of paths to kube config files.",
		},
		"config_path": schema.StringAttribute{
			Computed:    true,
			Description: "Path to the kube config file.",
		},
		"config_context": schema.StringAttribute{
			Computed:    true,
			Description: "Context name to load from the kube config file.",
		},
		"config_context_auth_info": schema.StringAttribute{
			Computed:    true,
			Description: "",
		},
		"config_context_cluster": schema.StringAttribute{
			Computed:    true,
			Description: "",
		},
		"token": schema.StringAttribute{
			Computed:    true,
			Sensitive:   true,
			Description: "Token to authenticate an service account",
		},
		"proxy_url": schema.StringAttribute{
			Computed:    true,
			Description: "URL to the proxy to be used for all API requests",
		},
	}
}

package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *AkpInstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: getAKPInstanceAttributes(),
	}
}

func getAKPInstanceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Instance ID",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"argocd": schema.SingleNestedAttribute{
			Required:   true,
			Attributes: getArgoCDAttributes(),
		},
		"argocd_cm": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: getConfigMapAttributes(),
		},
		"argocd_rbac_cm": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: getConfigMapAttributes(),
		},
		"argocd_secret": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: getSecretAttributes(),
		},
		"argocd_notifications_cm": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: getConfigMapAttributes(),
		},
		"argocd_notifications_secret": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: getSecretAttributes(),
		},
		"argocd_image_updater_config": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: getConfigMapAttributes(),
		},
		"argocd_image_updater_ssh_config": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: getConfigMapAttributes(),
		},
		"argocd_image_updater_secret": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: getSecretAttributes(),
		},
		"clusters": schema.ListNestedAttribute{
			Optional:  true,
			Sensitive: false,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getClusterAttributes(),
			},
		},
		"argocd_ssh_known_hosts_cm": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: getConfigMapAttributes(),
		},
		"argocd_tls_certs_cm": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: getConfigMapAttributes(),
		},
		"repo_credential_secrets": schema.ListNestedAttribute{
			Optional: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getSecretAttributes(),
			},
		},
		"repo_template_credential_secrets": schema.ListNestedAttribute{
			Optional: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getSecretAttributes(),
			},
		},
	}
}

func getConfigMapAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"data": schema.MapAttribute{
			ElementType: types.StringType,
			Optional:    true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getSecretAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Name",
		},
		"labels": schema.MapAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "Labels",
			Optional:            true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
		"data": schema.MapAttribute{
			ElementType: types.StringType,
			Optional:    true,
			Sensitive:   true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
		"string_data": schema.MapAttribute{
			ElementType: types.StringType,
			Optional:    true,
			Computed:    true,
			Sensitive:   true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
		"type": schema.StringAttribute{
			Optional: true,
		},
	}
}

func getArgoCDAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Name",
		},
		"spec": schema.SingleNestedAttribute{
			Required:   true,
			Attributes: getArgoCDSpecAttributes(),
		},
	}
}

func getClusterAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Name",
		},
		"namespace": schema.StringAttribute{
			MarkdownDescription: "Agent Installation Namespace",
			Required:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"labels": schema.MapAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "Labels",
			Optional:            true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
		"annotations": schema.MapAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "Annotations",
			Optional:            true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
		"spec": schema.SingleNestedAttribute{
			Required:   true,
			Attributes: getClusterSpecAttributes(),
		},
		"kubeconfig": schema.SingleNestedAttribute{
			MarkdownDescription: "Kubernetes connection settings. If configured, terraform will try to connect to the cluster and install the agent",
			Optional:            true,
			Attributes:          getKubeconfigAttributes(),
		},
		"manifests": schema.StringAttribute{
			MarkdownDescription: "Agent Installation Manifests",
			Computed:            true,
			Sensitive:           true,
		},
	}
}

func getArgoCDSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"description": schema.StringAttribute{
			Optional: true,
		},
		"version": schema.StringAttribute{
			Required: true,
		},
		"instance_spec": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: getInstanceSpecAttributes(),
		},
	}
}

func getInstanceSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ip_allow_list": schema.ListNestedAttribute{
			MarkdownDescription: "IP Allow List",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getIPAllowListEntryAttributes(),
			},
		},
		"subdomain": schema.StringAttribute{
			MarkdownDescription: "Instance Subdomain. By default equals to instance id",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"declarative_management_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Declarative Management",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"extensions": schema.ListNestedAttribute{
			MarkdownDescription: "Extensions",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getArgoCDExtensionInstallEntryAttributes(),
			},
		},
		"cluster_customization_defaults": schema.SingleNestedAttribute{
			MarkdownDescription: "Default Values For Cluster Agents",
			Optional:            true,
			Computed:            true,
			Attributes:          getClusterCustomizationAttributes(),
		},
		"image_updater_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Image Updater",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"backend_ip_allow_list_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable IP Allow List to Cluster Agents",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"repo_server_delegate": schema.SingleNestedAttribute{
			MarkdownDescription: "In case some clusters don't have network access to your private Git provider you can delegate these operations to one specific cluster.",
			Optional:            true,
			Attributes:          getRepoServerDelegateAttributes(),
		},
		"audit_extension_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Audit Extension. Set this to `true` to install Audit Extension to Argo CD instance. Do not use `spec.extensions` for that",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"sync_history_extension_enabled": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"image_updater_delegate": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: getImageUpdaterDelegateAttributes(),
		},
		"app_set_delegate": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: getAppSetDelegateAttributes(),
		},
	}
}

func getIPAllowListEntryAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ip": schema.StringAttribute{
			MarkdownDescription: "IP Address",
			Required:            true,
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "IP Description",
			Optional:            true,
		},
	}
}

func getArgoCDExtensionInstallEntryAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "Extension ID",
			Required:            true,
		},
		"version": schema.StringAttribute{
			MarkdownDescription: "Extension version",
			Required:            true,
		},
	}
}

func getClusterCustomizationAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"auto_upgrade_disabled": schema.BoolAttribute{
			MarkdownDescription: "Disable Agent Auto-upgrade",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"kustomization": schema.StringAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"app_replication": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"redis_tunneling": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getRepoServerDelegateAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"control_plane": schema.BoolAttribute{
			MarkdownDescription: "Use Control Plane",
			Required:            true,
		},
		"managed_cluster": schema.SingleNestedAttribute{
			MarkdownDescription: "Use Managed Cluster",
			Optional:            true,
			Attributes:          getManagedClusterAttributes(),
		},
	}
}

func getImageUpdaterDelegateAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"control_plane": schema.BoolAttribute{
			MarkdownDescription: "Use Control Plane",
			Required:            true,
		},
		"managed_cluster": schema.SingleNestedAttribute{
			MarkdownDescription: "Use Managed Cluster",
			Optional:            true,
			Attributes:          getManagedClusterAttributes(),
		},
	}
}

func getAppSetDelegateAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"managed_cluster": schema.SingleNestedAttribute{
			MarkdownDescription: "Use Managed Cluster",
			Optional:            true,
			Attributes:          getManagedClusterAttributes(),
		},
	}
}

func getManagedClusterAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"cluster_name": schema.StringAttribute{
			MarkdownDescription: "Cluster Name",
			Required:            true,
		},
	}
}

func getClusterSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"description": schema.StringAttribute{
			MarkdownDescription: "Cluster Description",
			Optional:            true,
		},
		"namespace_scoped": schema.BoolAttribute{
			MarkdownDescription: "Agent Namespace Scoped",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"data": schema.SingleNestedAttribute{
			Optional:   true,
			Attributes: getClusterDataAttributes(),
		},
	}
}

func getClusterDataAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"size": schema.StringAttribute{
			MarkdownDescription: "Cluster Size. One of `small`, `medium` or `large`",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"auto_upgrade_disabled": schema.BoolAttribute{
			MarkdownDescription: "Disable Agents Auto Upgrade. On resource update terraform will try to update the agent if this is set to `true`. Otherwise agent will update itself automatically",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"kustomization": schema.StringAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"app_replication": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"target_version": schema.StringAttribute{
			MarkdownDescription: "Installed agent version",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"redis_tunneling": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getKubeconfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"host": schema.StringAttribute{
			Optional:    true,
			Description: "The hostname (in form of URI) of Kubernetes master.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"username": schema.StringAttribute{
			Optional:    true,
			Description: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"password": schema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"insecure": schema.BoolAttribute{
			Optional:    true,
			Description: "Whether server should be accessed without verifying the TLS certificate.",
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"client_certificate": schema.StringAttribute{
			Optional:    true,
			Description: "PEM-encoded client certificate for TLS authentication.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"client_key": schema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "PEM-encoded client certificate key for TLS authentication.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"cluster_ca_certificate": schema.StringAttribute{
			Optional:    true,
			Description: "PEM-encoded root certificates bundle for TLS authentication.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"config_paths": schema.ListAttribute{
			ElementType: types.StringType,
			Optional:    true,
			Description: "A list of paths to kube config files.",
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
		},
		"config_path": schema.StringAttribute{
			Optional:    true,
			Description: "Path to the kube config file.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"config_context": schema.StringAttribute{
			Optional:    true,
			Description: "Context name to load from the kube config file.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"config_context_auth_info": schema.StringAttribute{
			Optional:    true,
			Description: "",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"config_context_cluster": schema.StringAttribute{
			Optional:    true,
			Description: "",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"token": schema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "Token to authenticate an service account",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"proxy_url": schema.StringAttribute{
			Optional:    true,
			Description: "URL to the proxy to be used for all API requests",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

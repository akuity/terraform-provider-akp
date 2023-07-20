package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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
		"name": schema.StringAttribute{
			Required:            true,
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
		"spec": schema.SingleNestedAttribute{
			Required:   true,
			Attributes: getArgoCDSpecAttributes(),
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

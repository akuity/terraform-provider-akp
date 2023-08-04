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
		MarkdownDescription: "Manages an Argo CD instance",
		Attributes:          getAKPInstanceAttributes(),
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
			MarkdownDescription: "Instance name",
		},
		"argocd": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD instance",
			Required:            true,
			Attributes:          getArgoCDAttributes(),
		},
		"argocd_cm": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD configmap",
			Optional:            true,
			Computed:            true,
			Attributes:          getConfigMapAttributes(),
		},
		"argocd_rbac_cm": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD rbac configmap",
			Optional:            true,
			Computed:            true,
			Attributes:          getConfigMapAttributes(),
		},
		"argocd_secret": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD secret",
			Optional:            true,
			Attributes:          getSecretAttributes(),
		},
		"argocd_notifications_cm": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD notifications configmap",
			Optional:            true,
			Computed:            true,
			Attributes:          getConfigMapAttributes(),
		},
		"argocd_notifications_secret": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD notifiations secret",
			Optional:            true,
			Attributes:          getSecretAttributes(),
		},
		"argocd_image_updater_config": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD image updater configmap",
			Optional:            true,
			Computed:            true,
			Attributes:          getConfigMapAttributes(),
		},
		"argocd_image_updater_ssh_config": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD image updater ssh configmap",
			Optional:            true,
			Computed:            true,
			Attributes:          getConfigMapAttributes(),
		},
		"argocd_image_updater_secret": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD image updater secret",
			Optional:            true,
			Attributes:          getSecretAttributes(),
		},
		"argocd_ssh_known_hosts_cm": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD ssh known hosts configmap",
			Optional:            true,
			Computed:            true,
			Attributes:          getConfigMapAttributes(),
		},
		"argocd_tls_certs_cm": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD tls certs configmap",
			Optional:            true,
			Computed:            true,
			Attributes:          getConfigMapAttributes(),
		},
		"repo_credential_secrets": schema.ListNestedAttribute{
			MarkdownDescription: "Argo CD repo credential secrets",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getSecretAttributes(),
			},
		},
		"repo_template_credential_secrets": schema.ListNestedAttribute{
			MarkdownDescription: "Argo CD repo templates credential secrets",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getSecretAttributes(),
			},
		},
	}
}

func getConfigMapAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"data": schema.MapAttribute{
			MarkdownDescription: "ConfigMap data",
			ElementType:         types.StringType,
			Required:            true,
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
			MarkdownDescription: "Secret name",
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
			MarkdownDescription: "Secret data",
			ElementType:         types.StringType,
			Optional:            true,
			Sensitive:           true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
		"string_data": schema.MapAttribute{
			MarkdownDescription: "Secret string data",
			ElementType:         types.StringType,
			Optional:            true,
			Sensitive:           true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
		"type": schema.StringAttribute{
			MarkdownDescription: "Secret type",
			Optional:            true,
		},
	}
}

func getArgoCDAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD instance spec",
			Required:            true,
			Attributes:          getArgoCDSpecAttributes(),
		},
	}
}

func getArgoCDSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"description": schema.StringAttribute{
			MarkdownDescription: "Instance description",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"version": schema.StringAttribute{
			MarkdownDescription: "Argo CD version. Should be equal to any [argo cd image tag](https://quay.io/repository/argoproj/argocd?tab=tags).",
			Required:            true,
		},
		"instance_spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD instance spec",
			Required:            true,
			Attributes:          getInstanceSpecAttributes(),
		},
	}
}

func getInstanceSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ip_allow_list": schema.ListNestedAttribute{
			MarkdownDescription: "IP allow list",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getIPAllowListEntryAttributes(),
			},
		},
		"subdomain": schema.StringAttribute{
			MarkdownDescription: "Instance subdomain. By default equals to instance id",
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
			MarkdownDescription: "Default values for cluster agents",
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
			MarkdownDescription: "Enable ip allow list for cluster agents",
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
			MarkdownDescription: "Enable Audit Extension. Set this to `true` to install Audit Extension to Argo CD instance.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"sync_history_extension_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Sync History Extension. Sync count and duration graphs as well as event details table on Argo CD application details page.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"image_updater_delegate": schema.SingleNestedAttribute{
			MarkdownDescription: "Select cluster in which you want to Install Image Updater",
			Optional:            true,
			Attributes:          getImageUpdaterDelegateAttributes(),
		},
		"app_set_delegate": schema.SingleNestedAttribute{
			MarkdownDescription: "Select cluster in which you want to Install Application Set controller",
			Optional:            true,
			Attributes:          getAppSetDelegateAttributes(),
		},
		"assistant_extension_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Powerful AI-powered assistant Extension. It helps analyze Kubernetes resources behavior and provides suggestions about resolving issues.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getIPAllowListEntryAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ip": schema.StringAttribute{
			MarkdownDescription: "IP address",
			Required:            true,
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "IP description",
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
			MarkdownDescription: "Disable Agents Auto Upgrade. On resource update terraform will try to update the agent if this is set to `true`. Otherwise agent will update itself automatically",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"kustomization": schema.StringAttribute{
			MarkdownDescription: "Kustomize configuration that will be applied to generated agent installation manifests",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"app_replication": schema.BoolAttribute{
			MarkdownDescription: "Enables Argo CD state replication to the managed cluster that allows disconnecting the cluster from Akuity Platform without losing core Argocd features",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"redis_tunneling": schema.BoolAttribute{
			MarkdownDescription: "Enables the ability to connect to Redis over a web-socket tunnel that allows using Akuity agent behind HTTPS proxy",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getRepoServerDelegateAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"control_plane": schema.BoolAttribute{
			MarkdownDescription: "If use control plane or not",
			Required:            true,
		},
		"managed_cluster": schema.SingleNestedAttribute{
			MarkdownDescription: "If use managed cluster or not",
			Optional:            true,
			Attributes:          getManagedClusterAttributes(),
		},
	}
}

func getImageUpdaterDelegateAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"control_plane": schema.BoolAttribute{
			MarkdownDescription: "If use control plane or not",
			Required:            true,
		},
		"managed_cluster": schema.SingleNestedAttribute{
			MarkdownDescription: "If use managed cluster or not",
			Optional:            true,
			Attributes:          getManagedClusterAttributes(),
		},
	}
}

func getAppSetDelegateAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"managed_cluster": schema.SingleNestedAttribute{
			MarkdownDescription: "Use managed cluster",
			Optional:            true,
			Attributes:          getManagedClusterAttributes(),
		},
	}
}

func getManagedClusterAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"cluster_name": schema.StringAttribute{
			MarkdownDescription: "Cluster name",
			Required:            true,
		},
	}
}

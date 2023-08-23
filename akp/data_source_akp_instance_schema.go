package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *AkpInstanceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets information about an Argo CD instance by its name",
		Attributes:          getAKPInstanceDataSourceAttributes(),
	}
}

func getAKPInstanceDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Instance ID",
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Instance name",
		},
		"argocd": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD instance",
			Computed:            true,
			Attributes:          getArgoCDDataSourceAttributes(),
		},
		"argocd_cm": schema.MapAttribute{
			MarkdownDescription: "Argo CD configmap",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_rbac_cm": schema.MapAttribute{
			MarkdownDescription: "Argo CD rbac configmap",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_secret": schema.MapAttribute{
			MarkdownDescription: "Argo CD secret",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_notifications_cm": schema.MapAttribute{
			MarkdownDescription: "Argo CD notifications configmap",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_notifications_secret": schema.MapAttribute{
			MarkdownDescription: "Argo CD notifiations secret",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_image_updater_config": schema.MapAttribute{
			MarkdownDescription: "Argo CD image updater configmap",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_image_updater_ssh_config": schema.MapAttribute{
			MarkdownDescription: "Argo CD image updater ssh configmap",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_image_updater_secret": schema.MapAttribute{
			MarkdownDescription: "Argo CD image updater secret",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_ssh_known_hosts_cm": schema.MapAttribute{
			MarkdownDescription: "Argo CD ssh known hosts configmap",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_tls_certs_cm": schema.MapAttribute{
			MarkdownDescription: "Argo CD tls certs configmap",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"repo_credential_secrets": schema.MapAttribute{
			MarkdownDescription: "Argo CD repo credential secrets",
			Computed:            true,
			ElementType:         types.MapType{ElemType: types.StringType},
		},
		"repo_template_credential_secrets": schema.MapAttribute{
			MarkdownDescription: "Argo CD repo templates credential secrets",
			Computed:            true,
			ElementType:         types.MapType{ElemType: types.StringType},
		},
	}
}

func getArgoCDDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD instance spec",
			Computed:            true,
			Attributes:          getArgoCDSpecDataSourceAttributes(),
		},
	}
}

func getArgoCDSpecDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"description": schema.StringAttribute{
			MarkdownDescription: "Instance description",
			Computed:            true,
		},
		"version": schema.StringAttribute{
			MarkdownDescription: "Argo CD version. Should be equal to any [argo cd image tag](https://quay.io/repository/argoproj/argocd?tab=tags).",
			Computed:            true,
		},
		"instance_spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD instance spec",
			Computed:            true,
			Attributes:          getInstanceSpecDataSourceAttributes(),
		},
	}
}

func getInstanceSpecDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ip_allow_list": schema.ListNestedAttribute{
			MarkdownDescription: "IP allow list",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getIPAllowListEntryDataSourceAttributes(),
			},
		},
		"subdomain": schema.StringAttribute{
			MarkdownDescription: "Instance subdomain. By default equals to instance id",
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
			MarkdownDescription: "Default values for cluster agents",
			Computed:            true,
			Attributes:          getClusterCustomizationDataSourceAttributes(),
		},
		"image_updater_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Image Updater",
			Computed:            true,
		},
		"backend_ip_allow_list_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable ip allow list for cluster agents",
			Computed:            true,
		},
		"repo_server_delegate": schema.SingleNestedAttribute{
			MarkdownDescription: "In case some clusters don't have network access to your private Git provider you can delegate these operations to one specific cluster.",
			Computed:            true,
			Attributes:          getRepoServerDelegateDataSourceAttributes(),
		},
		"audit_extension_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Audit Extension. Set this to `true` to install Audit Extension to Argo CD instance.",
			Computed:            true,
		},
		"sync_history_extension_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Sync History Extension. Sync count and duration graphs as well as event details table on Argo CD application details page.",
			Computed:            true,
		},
		"image_updater_delegate": schema.SingleNestedAttribute{
			MarkdownDescription: "Select cluster in which you want to Install Image Updater",
			Computed:            true,
			Attributes:          getImageUpdaterDelegateDataSourceAttributes(),
		},
		"app_set_delegate": schema.SingleNestedAttribute{
			MarkdownDescription: "Select cluster in which you want to Install Application Set controller",
			Computed:            true,
			Attributes:          getAppSetDelegateDataSourceAttributes(),
		},
		"assistant_extension_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Powerful AI-powered assistant Extension. It helps analyze Kubernetes resources behavior and provides suggestions about resolving issues.",
			Computed:            true,
		},
		"appset_policy": schema.SingleNestedAttribute{
			MarkdownDescription: "Configures Application Set policy settings.",
			Computed:            true,
			Attributes:          getAppsetPolicyDataSourceAttributes(),
		},
	}
}

func getAppsetPolicyDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"policy": schema.StringAttribute{
			MarkdownDescription: "Policy restricts what types of modifications will be made to managed Argo CD `Application` resources.\nAvailable options: `sync`, `create-only`, `create-delete`, and `create-update`.\n  - Policy `sync`(default): Update and delete are allowed.\n  - Policy `create-only`: Prevents ApplicationSet controller from modifying or deleting Applications.\n  - Policy `create-update`: Prevents ApplicationSet controller from deleting Applications. Update is allowed.\n  - Policy `create-delete`: Prevents ApplicationSet controller from modifying Applications, Delete is allowed.",
			Computed:            true,
		},
		"override_policy": schema.BoolAttribute{
			MarkdownDescription: "Allows per `ApplicationSet` sync policy.",
			Computed:            true,
		},
	}
}

func getIPAllowListEntryDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ip": schema.StringAttribute{
			MarkdownDescription: "IP address",
			Computed:            true,
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "IP description",
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
			MarkdownDescription: "Disable Agents Auto Upgrade. On resource update terraform will try to update the agent if this is set to `true`. Otherwise agent will update itself automatically",
			Computed:            true,
		},
		"kustomization": schema.StringAttribute{
			MarkdownDescription: "Kustomize configuration that will be applied to generated agent installation manifests",
			Computed:            true,
		},
		"app_replication": schema.BoolAttribute{
			MarkdownDescription: "Enables Argo CD state replication to the managed cluster that allows disconnecting the cluster from Akuity Platform without losing core Argocd features",
			Computed:            true,
		},
		"redis_tunneling": schema.BoolAttribute{
			MarkdownDescription: "Enables the ability to connect to Redis over a web-socket tunnel that allows using Akuity agent behind HTTPS proxy",
			Computed:            true,
		},
	}
}

func getRepoServerDelegateDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"control_plane": schema.BoolAttribute{
			MarkdownDescription: "If use control plane or not",
			Computed:            true,
		},
		"managed_cluster": schema.SingleNestedAttribute{
			MarkdownDescription: "If use managed cluster or not",
			Computed:            true,
			Attributes:          getManagedClusterDataSourceAttributes(),
		},
	}
}

func getImageUpdaterDelegateDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"control_plane": schema.BoolAttribute{
			MarkdownDescription: "If use control plane or not",
			Computed:            true,
		},
		"managed_cluster": schema.SingleNestedAttribute{
			MarkdownDescription: "If use managed cluster or not",
			Computed:            true,
			Attributes:          getManagedClusterDataSourceAttributes(),
		},
	}
}

func getAppSetDelegateDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"managed_cluster": schema.SingleNestedAttribute{
			MarkdownDescription: "Use managed cluster",
			Computed:            true,
			Attributes:          getManagedClusterDataSourceAttributes(),
		},
	}
}

func getManagedClusterDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"cluster_name": schema.StringAttribute{
			MarkdownDescription: "Cluster name",
			Computed:            true,
		},
	}
}

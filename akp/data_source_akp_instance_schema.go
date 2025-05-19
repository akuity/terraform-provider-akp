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
			MarkdownDescription: "is aligned with the options in `argocd-cm` ConfigMap as described in the [ArgoCD Atomic Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#atomic-configuration). For a concrete example, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-cm-yaml/).",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_rbac_cm": schema.MapAttribute{
			MarkdownDescription: "is aligned with the options in `argocd-rbac-cm` ConfigMap as described in the [ArgoCD Atomic Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#atomic-configuration). For a concrete example, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-rbac-cm-yaml/).",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_secret": schema.MapAttribute{
			MarkdownDescription: "is aligned with the options in `argocd-secret` Secret as described in the [ArgoCD Atomic Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#atomic-configuration). For a concrete example, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-secret-yaml/).",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"application_set_secret": schema.MapAttribute{
			MarkdownDescription: "stores secret key-value that will be used by `ApplicationSet`. For an example of how to use this in your ApplicationSet's pull request generator, see [here](https://github.com/argoproj/argo-cd/blob/master/docs/operator-manual/applicationset/Generators-Pull-Request.md#github). In this example, `tokenRef.secretName` would be application-set-secret.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_notifications_cm": schema.MapAttribute{
			MarkdownDescription: "configures Argo CD notifications, and it is aligned with `argocd-notifications-cm` ConfigMap of Argo CD, for more details and examples, refer to [this documentation](https://argocd-notifications.readthedocs.io/en/stable/).",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_notifications_secret": schema.MapAttribute{
			MarkdownDescription: "contains sensitive data of Argo CD notifications, and it is aligned with `argocd-notifications-secret` Secret of Argo CD, for more details and examples, refer to [this documentation](https://argocd-notifications.readthedocs.io/en/stable/services/overview/#sensitive-data).",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_image_updater_config": schema.MapAttribute{
			MarkdownDescription: "configures Argo CD image updater, and it is aligned with `argocd-image-updater-config` ConfigMap of Argo CD, for available options and examples, refer to [this documentation](https://argocd-image-updater.readthedocs.io/en/stable/).",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_image_updater_ssh_config": schema.MapAttribute{
			MarkdownDescription: "contains the ssh configuration for Argo CD image updater, and it is aligned with `argocd-image-updater-ssh-config` ConfigMap of Argo CD, for available options and examples, refer to [this documentation](https://argocd-image-updater.readthedocs.io/en/stable/).",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_image_updater_secret": schema.MapAttribute{
			MarkdownDescription: "contains sensitive data (e.g., credentials for image updater to access registries) of Argo CD image updater, for available options and examples, refer to [this documentation](https://argocd-image-updater.readthedocs.io/en/stable/).",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_ssh_known_hosts_cm": schema.MapAttribute{
			MarkdownDescription: "is aligned with the options in `argocd-ssh-known-hosts-cm` ConfigMap as described in the [ArgoCD Atomic Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#atomic-configuration). For a concrete example, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-ssh-known-hosts-cm-yaml/).",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"argocd_tls_certs_cm": schema.MapAttribute{
			MarkdownDescription: "is aligned with the options in `argocd-tls-certs-cm` ConfigMap as described in the [ArgoCD Atomic Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#atomic-configuration). For a concrete example, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-tls-certs-cm-yaml/).",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"repo_credential_secrets": schema.MapAttribute{
			MarkdownDescription: "is a map of repo credential secrets, the key of map entry is the `name` of the secret, and the value is the aligned with options in `argocd-repositories.yaml.data` as described in the [ArgoCD Atomic Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#atomic-configuration). For a concrete example, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-repositories-yaml/).",
			Computed:            true,
			ElementType:         types.MapType{ElemType: types.StringType},
		},
		"repo_template_credential_secrets": schema.MapAttribute{
			MarkdownDescription: "is a map of repository credential templates secrets, the key of map entry is the `name` of the secret, and the value is the aligned with options in `argocd-repo-creds.yaml.data` as described in the [ArgoCD Atomic Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#atomic-configuration). For a concrete example, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-repo-creds.yaml/).",
			Computed:            true,
			ElementType:         types.MapType{ElemType: types.StringType},
		},
		"config_management_plugins": schema.MapNestedAttribute{
			MarkdownDescription: "is a map of [Config Management Plugins](https://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/#config-management-plugins), the key of map entry is the `name` of the plugin, and the value is the definition of the Config Management Plugin(v2).",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getAKPConfigManagementPluginDataSourceAttributes(),
			},
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
			MarkdownDescription: "Argo CD version. Should be equal to any Akuity [`argocd` image tag](https://quay.io/repository/akuity/argocd?tab=tags).",
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
		"host_aliases": schema.ListNestedAttribute{
			MarkdownDescription: "Host Aliases that override the DNS entries for control plane Argo CD components such as API Server and Dex.",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getHostAliasesDataSourceAttributes(),
			},
		},
		"crossplane_extension": schema.SingleNestedAttribute{
			MarkdownDescription: "Custom Resource Definition group name that identifies the Crossplane resource in kubernetes. We will include built-in crossplane resources. Note that you can use glob pattern to match the group. ie. *.crossplane.io",
			Computed:            true,
			Attributes:          getCrossplaneExtensionDataSourceAttributes(),
		},
		"agent_permissions_rules": schema.ListNestedAttribute{
			MarkdownDescription: "The ability to configure agent permissions rules.",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getAgentPermissionsRuleDataSourceAttributes(),
			},
		},
		"fqdn": schema.StringAttribute{
			MarkdownDescription: "Configures the FQDN for the argocd instance, for ingress URL, domain suffix, etc.",
			Computed:            true,
		},
		"multi_cluster_k8s_dashboard_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable the KubeVision feature",
			Computed:            true,
		},
		"app_in_any_namespace_config": schema.SingleNestedAttribute{
			MarkdownDescription: "App in any namespace config",
			Computed:            true,
			Attributes:          getAppInAnyNamespaceConfigDataSourceAttributes(),
		},
		"appset_plugins": schema.ListNestedAttribute{
			MarkdownDescription: "Application Set plugins",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getAppsetPluginsDataSourceAttributes(),
			},
		},
	}
}

func getAppsetPluginsDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			MarkdownDescription: "Plugin name",
			Computed:            true,
		},
		"token": schema.StringAttribute{
			MarkdownDescription: "Plugin token",
			Computed:            true,
		},
		"base_url": schema.StringAttribute{
			MarkdownDescription: "Plugin base URL",
			Computed:            true,
		},
		"request_timeout": schema.Int64Attribute{
			MarkdownDescription: "Plugin request timeout",
			Computed:            true,
		},
	}
}

func getHostAliasesDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ip": schema.StringAttribute{
			MarkdownDescription: "IP address",
			Computed:            true,
		},
		"hostnames": schema.ListAttribute{
			MarkdownDescription: "Hostnames",
			Computed:            true,
			ElementType:         types.StringType,
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

func getAKPConfigManagementPluginDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether this plugin is enabled or not. Default to false.",
			Computed:            true,
		},
		"image": schema.StringAttribute{
			MarkdownDescription: "Image to use for the plugin",
			Computed:            true,
		},
		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Plugin spec",
			Computed:            true,
			Attributes:          getPluginSpecDataSourceAttributes(),
		},
	}
}

func getPluginSpecDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"version": schema.StringAttribute{
			MarkdownDescription: "Plugin version",
			Computed:            true,
		},
		"init": schema.SingleNestedAttribute{
			MarkdownDescription: "The init command runs in the Application source directory at the beginning of each manifest generation. The init command can output anything. A non-zero status code will fail manifest generation. Init always happens immediately before generate, but its output is not treated as manifests. This is a good place to, for example, download chart dependencies.",
			Computed:            true,
			Attributes:          getCommandDataSourceAttributes(),
		},
		"generate": schema.SingleNestedAttribute{
			MarkdownDescription: "The generate command runs in the Application source directory each time manifests are generated. Standard output must be ONLY valid Kubernetes Objects in either YAML or JSON. A non-zero exit code will fail manifest generation. Error output will be sent to the UI, so avoid printing sensitive information (such as secrets).",
			Computed:            true,
			Attributes:          getCommandDataSourceAttributes(),
		},
		"discover": schema.SingleNestedAttribute{
			MarkdownDescription: "The discovery config is applied to a repository. If every configured discovery tool matches, then the plugin may be used to generate manifests for Applications using the repository. If the discovery config is omitted then the plugin will not match any application but can still be invoked explicitly by specifying the plugin name in the app spec. Only one of fileName, find.glob, or find.command should be specified. If multiple are specified then only the first (in that order) is evaluated.",
			Computed:            true,
			Attributes:          getDiscoverDataSourceAttributes(),
		},
		"parameters": schema.SingleNestedAttribute{
			MarkdownDescription: "The parameters config describes what parameters the UI should display for an Application. It is up to the user to actually set parameters in the Application manifest (in spec.source.plugin.parameters). The announcements only inform the \"Parameters\" tab in the App Details page of the UI.",
			Computed:            true,
			Attributes:          getParametersDataSourceAttributes(),
		},
		"preserve_file_mode": schema.BoolAttribute{
			MarkdownDescription: "Whether the plugin receives repository files with original file mode. Dangerous since the repository might have executable files. Set to true only if you trust the CMP plugin authors. Set to false by default.",
			Computed:            true,
		},
	}
}

func getCommandDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"command": schema.ListAttribute{
			MarkdownDescription: "Command",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"args": schema.ListAttribute{
			MarkdownDescription: "Arguments of the command",
			Computed:            true,
			ElementType:         types.StringType,
		},
	}
}

func getDiscoverDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"find": schema.SingleNestedAttribute{
			MarkdownDescription: "Find config",
			Computed:            true,
			Attributes:          getFindDataSourceAttributes(),
		},
		"file_name": schema.StringAttribute{
			MarkdownDescription: "A glob pattern (https://pkg.go.dev/path/filepath#Glob) that is applied to the Application's source directory. If there is a match, this plugin may be used for the Application.",
			Computed:            true,
		},
	}
}

func getFindDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"command": schema.ListAttribute{
			MarkdownDescription: "The find command runs in the repository's root directory. To match, it must exit with status code 0 and produce non-empty output to standard out.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"args": schema.ListAttribute{
			MarkdownDescription: "Arguments for the find command",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"glob": schema.StringAttribute{
			MarkdownDescription: "This does the same thing as `file_name`, but it supports double-start (nested directory) glob patterns.",
			Computed:            true,
		},
	}
}

func getParametersDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"static": schema.ListNestedAttribute{
			MarkdownDescription: "Static parameter announcements are sent to the UI for all Applications handled by this plugin. Think of the `string`, `array`, and `map` values set here as defaults. It is up to the plugin author to make sure that these default values actually reflect the plugin's behavior if the user doesn't explicitly set different values for those parameters.",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getParameterAnnouncementDataSourceAttributes(),
			},
		},
		"dynamic": schema.SingleNestedAttribute{
			MarkdownDescription: "Dynamic parameter announcements are announcements specific to an Application handled by this plugin. For example, the values for a Helm chart's values.yaml file could be sent as parameter announcements.",
			Computed:            true,
			Attributes:          getDynamicDataSourceAttributes(),
		},
	}
}

func getParameterAnnouncementDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			MarkdownDescription: "Parameter name",
			Computed:            true,
		},
		"title": schema.StringAttribute{
			MarkdownDescription: "Title and description of the parameter",
			Computed:            true,
		},
		"tooltip": schema.StringAttribute{
			MarkdownDescription: "Tooltip of the Parameter, will be shown when hovering over the title",
			Computed:            true,
		},
		"required": schema.BoolAttribute{
			MarkdownDescription: "Whether the Parameter is required or not. If this field is set to true, the UI will indicate to the user that they must set the value. Default to false.",
			Computed:            true,
		},
		"item_type": schema.StringAttribute{
			MarkdownDescription: "Item type tells the UI how to present the parameter's value (or, for arrays and maps, values). Default is `string`. Examples of other types which may be supported in the future are `boolean` or `number`. Even if the itemType is not `string`, the parameter value from the Application spec will be sent to the plugin as a string. It's up to the plugin to do the appropriate conversion.",
			Computed:            true,
		},
		"collection_type": schema.StringAttribute{
			MarkdownDescription: "Collection Type describes what type of value this parameter accepts (string, array, or map) and allows the UI to present a form to match that type. Default is `string`. This field must be present for non-string types. It will not be inferred from the presence of an `array` or `map` field.",
			Computed:            true,
		},
		"string": schema.StringAttribute{
			MarkdownDescription: "This field communicates the parameter's default value to the UI if the parameter is a `string`.",
			Computed:            true,
		},
		"array": schema.ListAttribute{
			MarkdownDescription: "This field communicates the parameter's default value to the UI if the parameter is an `array`.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"map": schema.MapAttribute{
			MarkdownDescription: "This field communicates the parameter's default value to the UI if the parameter is a `map`.",
			Computed:            true,
			ElementType:         types.StringType,
		},
	}
}

func getDynamicDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"command": schema.ListAttribute{
			MarkdownDescription: "The command will run in an Application's source directory. Standard output must be JSON matching the schema of the static parameter announcements list.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"args": schema.ListAttribute{
			MarkdownDescription: "Arguments of the command",
			Computed:            true,
			ElementType:         types.StringType,
		},
	}
}

func getCrossplaneExtensionDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"resources": schema.ListNestedAttribute{
			MarkdownDescription: "Glob patterns of the resources to match.",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getCrossplaneExtensionResourcesDataSourceAttributes(),
			},
		},
	}
}

func getCrossplaneExtensionResourcesDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"group": schema.StringAttribute{
			MarkdownDescription: "Glob pattern of the group to match.",
			Computed:            true,
		},
	}
}

func getAgentPermissionsRuleDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"api_groups": schema.ListAttribute{
			MarkdownDescription: "API groups of the rule.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"resources": schema.ListAttribute{
			MarkdownDescription: "Resources of the rule.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"verbs": schema.ListAttribute{
			MarkdownDescription: "Verbs of the rule.",
			Computed:            true,
			ElementType:         types.StringType,
		},
	}
}

func getAppInAnyNamespaceConfigDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether the app in any namespace config is enabled or not.",
			Computed:            true,
		},
	}
}

package akp

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	listplanmodifier2 "github.com/akuity/terraform-provider-akp/akp/modifiers/list"
	mapplanmodifier2 "github.com/akuity/terraform-provider-akp/akp/modifiers/map"
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
			Validators: []validator.String{
				stringvalidator.LengthBetween(minInstanceNameLength, maxInstanceNameLength),
				stringvalidator.RegexMatches(resourceNameRegex, resourceNameRegexDescription),
			},
		},
		"argocd": schema.SingleNestedAttribute{
			MarkdownDescription: "Argo CD instance configuration",
			Required:            true,
			Attributes:          getArgoCDAttributes(),
		},
		"argocd_cm": schema.MapAttribute{
			MarkdownDescription: "is aligned with the options in `argocd-cm` ConfigMap as described in the [ArgoCD Atomic Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#atomic-configuration). For a concrete example, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-cm-yaml/).",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
		"argocd_rbac_cm": schema.MapAttribute{
			MarkdownDescription: "is aligned with the options in `argocd-rbac-cm` ConfigMap as described in the [ArgoCD Atomic Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#atomic-configuration). For a concrete example, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-rbac-cm-yaml/).",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
		"argocd_secret": schema.MapAttribute{
			MarkdownDescription: "is aligned with the options in `argocd-secret` Secret as described in the [ArgoCD Atomic Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#atomic-configuration). For a concrete example, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-secret-yaml/).",
			Optional:            true,
			Sensitive:           true,
			ElementType:         types.StringType,
		},
		"application_set_secret": schema.MapAttribute{
			MarkdownDescription: "stores secret key-value that will be used by `ApplicationSet`. For an example of how to use this in your ApplicationSet's pull request generator, see [here](https://github.com/argoproj/argo-cd/blob/master/docs/operator-manual/applicationset/Generators-Pull-Request.md#github). In this example, `tokenRef.secretName` would be application-set-secret.",
			Optional:            true,
			Sensitive:           true,
			ElementType:         types.StringType,
		},
		"argocd_notifications_cm": schema.MapAttribute{
			MarkdownDescription: "configures Argo CD notifications, and it is aligned with `argocd-notifications-cm` ConfigMap of Argo CD, for more details and examples, refer to [this documentation](https://argo-cd.readthedocs.io/en/latest/operator-manual/notifications/).",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier2.UseStateForNullUnknown(),
			},
		},
		"argocd_notifications_secret": schema.MapAttribute{
			MarkdownDescription: "contains sensitive data of Argo CD notifications, and it is aligned with `argocd-notifications-secret` Secret of Argo CD, for more details and examples, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/notifications/templates/#defining-and-using-secrets-within-notification-templates).",
			Optional:            true,
			Sensitive:           true,
			ElementType:         types.StringType,
		},
		"argocd_image_updater_config": schema.MapAttribute{
			MarkdownDescription: "configures Argo CD image updater, and it is aligned with `argocd-image-updater-config` ConfigMap of Argo CD, for available options and examples, refer to [this documentation](https://argocd-image-updater.readthedocs.io/en/stable/).",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier2.UseStateForNullUnknown(),
			},
		},
		"argocd_image_updater_ssh_config": schema.MapAttribute{
			MarkdownDescription: "contains the ssh configuration for Argo CD image updater, and it is aligned with `argocd-image-updater-ssh-config` ConfigMap of Argo CD, for available options and examples, refer to [this documentation](https://argocd-image-updater.readthedocs.io/en/stable/).",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier2.UseStateForNullUnknown(),
			},
		},
		"argocd_image_updater_secret": schema.MapAttribute{
			MarkdownDescription: "contains sensitive data (e.g., credentials for image updater to access registries) of Argo CD image updater, for available options and examples, refer to [this documentation](https://argocd-image-updater.readthedocs.io/en/stable/).",
			Optional:            true,
			Sensitive:           true,
			ElementType:         types.StringType,
		},
		"argocd_ssh_known_hosts_cm": schema.MapAttribute{
			MarkdownDescription: "is aligned with the options in `argocd-ssh-known-hosts-cm` ConfigMap as described in the [ArgoCD Atomic Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#atomic-configuration). For a concrete example, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-ssh-known-hosts-cm-yaml/).",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
		"argocd_tls_certs_cm": schema.MapAttribute{
			MarkdownDescription: "is aligned with the options in `argocd-tls-certs-cm` ConfigMap as described in the [ArgoCD Atomic Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#atomic-configuration). For a concrete example, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-tls-certs-cm-yaml/).",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
		"repo_credential_secrets": schema.MapAttribute{
			MarkdownDescription: "is a map of repo credential secrets, the key of map entry is the `name` of the secret, and the value is the aligned with options in `argocd-repositories.yaml.data` as described in the [ArgoCD Atomic Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#atomic-configuration). For a concrete example, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-repositories-yaml/).",
			Optional:            true,
			Sensitive:           true,
			ElementType:         types.MapType{ElemType: types.StringType},
			Validators: []validator.Map{
				mapvalidator.KeysAre(stringvalidator.RegexMatches(regexp.MustCompile("repo-.+"), "invalid secret name, repo credential secret name should start with 'repo-'")),
			},
		},
		"repo_template_credential_secrets": schema.MapAttribute{
			MarkdownDescription: "is a map of repository credential templates secrets, the key of map entry is the `name` of the secret, and the value is the aligned with options in `argocd-repo-creds.yaml.data` as described in the [ArgoCD Atomic Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#atomic-configuration). For a concrete example, refer to [this documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-repo-creds.yaml/).",
			Optional:            true,
			Sensitive:           true,
			ElementType:         types.MapType{ElemType: types.StringType},
			Validators: []validator.Map{
				mapvalidator.KeysAre(stringvalidator.RegexMatches(regexp.MustCompile("repo-.+"), "invalid secret name, repo template credential secret name should start with 'repo-'")),
			},
		},
		"config_management_plugins": schema.MapNestedAttribute{
			MarkdownDescription: "is a map of [Config Management Plugins](https://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/#config-management-plugins), the key of map entry is the `name` of the plugin, and the value is the definition of the Config Management Plugin(v2).",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getAKPConfigManagementPluginAttributes(),
			},
		},
		"argocd_resources": schema.MapAttribute{
			MarkdownDescription: "Map of ArgoCD custom resources to be managed alongside the ArgoCD instance. Currently supported resources are: `ApplicationSet`, `Application`, `AppProject`. Should all be in the apiVersion `argoproj.io/v1alpha1`.",
			Optional:            true,
			ElementType:         types.StringType,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
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
			MarkdownDescription: "Argo CD version. Should be equal to any Akuity [`argocd` image tag](https://quay.io/repository/akuity/argocd?tab=tags).",
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
			MarkdownDescription: "**Deprecated:** Use the `akp_instance_ip_allow_list` resource instead. IP allow list for instance access. When not specified in the configuration, this field will not be managed by this resource, allowing `akp_instance_ip_allow_list` resources to manage it independently.",
			Optional:            true,
			Computed:            true,
			DeprecationMessage:  "Use the akp_instance_ip_allow_list resource for managing IP allow lists. Remove this field from your configuration to allow akp_instance_ip_allow_list resources to manage the IP allow list independently.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: getIPAllowListEntryAttributes(),
			},
			PlanModifiers: []planmodifier.List{
				listplanmodifier2.IgnoreWhenNotConfigured(),
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
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getArgoCDExtensionInstallEntryAttributes(),
			},
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
		},
		"cluster_customization_defaults": schema.SingleNestedAttribute{
			MarkdownDescription: "Default values for cluster agents",
			Optional:            true,
			Computed:            true,
			Attributes:          getClusterCustomizationAttributes(),
			PlanModifiers: []planmodifier.Object{
				objectplanmodifier.UseStateForUnknown(),
			},
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
			MarkdownDescription: "Deprecated: upcoming removal. Enable Powerful AI-powered assistant Extension. It helps analyze Kubernetes resources behavior and provides suggestions about resolving issues.",
			Optional:            true,
			Computed:            true,
			DeprecationMessage:  "assistant_extension_enabled field will be removed in a future release; remove it from configs.",
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"application_set_extension": schema.SingleNestedAttribute{
			MarkdownDescription: "Configuration for the ApplicationSet extension, enabling the controller and UI integration within Argo CD.",
			Optional:            true,
			Attributes:          getApplicationSetExtensionAttributes(),
		},
		"appset_policy": schema.SingleNestedAttribute{
			MarkdownDescription: "Configures Application Set policy settings.",
			Optional:            true,
			Computed:            true,
			Attributes:          getAppsetPolicyAttributes(),
			PlanModifiers: []planmodifier.Object{
				objectplanmodifier.UseStateForUnknown(),
			},
		},
		"host_aliases": schema.ListNestedAttribute{
			MarkdownDescription: "Host Aliases that override the DNS entries for control plane Argo CD components such as API Server and Dex.",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getHostAliasAttributes(),
			},
		},
		"crossplane_extension": schema.SingleNestedAttribute{
			MarkdownDescription: "Custom Resource Definition group name that identifies the Crossplane resource in kubernetes. We will include built-in crossplane resources. Note that you can use glob pattern to match the group. ie. *.crossplane.io",
			Optional:            true,
			Attributes:          getCrossplaneExtensionAttributes(),
		},
		"agent_permissions_rules": schema.ListNestedAttribute{
			MarkdownDescription: "The ability to configure agent permissions rules.",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getAgentPermissionsRuleAttributes(),
			},
		},
		"fqdn": schema.StringAttribute{
			MarkdownDescription: "Configures the FQDN for the argocd instance, for ingress URL, domain suffix, etc.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"multi_cluster_k8s_dashboard_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable the KubeVision feature",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"app_in_any_namespace_config": schema.SingleNestedAttribute{
			MarkdownDescription: "App in any namespace config",
			Optional:            true,
			Attributes:          getAppInAnyNamespaceConfigAttributes(),
		},
		"appset_plugins": schema.ListNestedAttribute{
			MarkdownDescription: "Application Set plugins",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getAppsetPluginsAttributes(),
			},
		},
		"akuity_intelligence_extension": schema.SingleNestedAttribute{
			MarkdownDescription: "Akuity Intelligence Extension configuration for enhanced AI-powered features",
			Optional:            true,
			Attributes:          getAkuityIntelligenceExtensionAttributes(),
		},
		"cluster_addons_extension": schema.SingleNestedAttribute{
			MarkdownDescription: "Cluster Addons Extension configuration for managing cluster addons",
			Optional:            true,
			Attributes:          getClusterAddonsExtensionAttributes(),
		},
		"kube_vision_config": schema.SingleNestedAttribute{
			MarkdownDescription: "Advanced Akuity Intelligence configuration like CVE scanning and AI runbooks",
			Optional:            true,
			Attributes:          getKubeVisionConfigAttributes(),
		},
		"metrics_ingress_username": schema.StringAttribute{
			MarkdownDescription: "Username for metrics ingress authentication",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"metrics_ingress_password_hash": schema.StringAttribute{
			MarkdownDescription: "Password hash for metrics ingress authentication",
			Optional:            true,
			Sensitive:           true,
		},
		"privileged_notification_cluster": schema.StringAttribute{
			MarkdownDescription: "Cluster name where notifications controller will be installed with elevated privileges to see controlplane and intg. cluster apps",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getAppsetPluginsAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			MarkdownDescription: "Plugin name",
			Required:            true,
		},
		"token": schema.StringAttribute{
			MarkdownDescription: "Plugin token",
			Required:            true,
		},
		"base_url": schema.StringAttribute{
			MarkdownDescription: "Plugin base URL",
			Required:            true,
		},
		"request_timeout": schema.Int64Attribute{
			MarkdownDescription: "Plugin request timeout",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getHostAliasAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"hostnames": schema.ListAttribute{
			MarkdownDescription: "List of hostnames",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"ip": schema.StringAttribute{
			MarkdownDescription: "IP address",
			Required:            true,
		},
	}
}

func getAppsetPolicyAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"policy": schema.StringAttribute{
			MarkdownDescription: "Policy restricts what types of modifications will be made to managed Argo CD `Application` resources.\nAvailable options: `sync`, `create-only`, `create-delete`, and `create-update`.\n  - Policy `sync`(default): Update and delete are allowed.\n  - Policy `create-only`: Prevents ApplicationSet controller from modifying or deleting Applications.\n  - Policy `create-update`: Prevents ApplicationSet controller from deleting Applications. Update is allowed.\n  - Policy `create-delete`: Prevents ApplicationSet controller from modifying Applications, Delete is allowed.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
			Validators: []validator.String{
				stringvalidator.OneOf("sync", "create-only", "create-update", "create-delete"),
			},
		},
		"override_policy": schema.BoolAttribute{
			MarkdownDescription: "Allows per `ApplicationSet` sync policy.",
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
		"server_side_diff_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enables the ability to set server-side diff on the application-controller.",
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

func getAKPConfigManagementPluginAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether this plugin is enabled or not. Default to false.",
			Computed:            true,
			Optional:            true,
			Default:             booldefault.StaticBool(false),
		},
		"image": schema.StringAttribute{
			MarkdownDescription: "Image to use for the plugin",
			Required:            true,
		},
		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Plugin spec",
			Required:            true,
			Attributes:          getPluginSpecAttributes(),
		},
	}
}

func getPluginSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"version": schema.StringAttribute{
			MarkdownDescription: "Plugin version",
			Optional:            true,
		},
		"init": schema.SingleNestedAttribute{
			MarkdownDescription: "The init command runs in the Application source directory at the beginning of each manifest generation. The init command can output anything. A non-zero status code will fail manifest generation. Init always happens immediately before generate, but its output is not treated as manifests. This is a good place to, for example, download chart dependencies.",
			Optional:            true,
			Attributes:          getCommandAttributes(),
		},
		"generate": schema.SingleNestedAttribute{
			MarkdownDescription: "The generate command runs in the Application source directory each time manifests are generated. Standard output must be ONLY valid Kubernetes Objects in either YAML or JSON. A non-zero exit code will fail manifest generation. Error output will be sent to the UI, so avoid printing sensitive information (such as secrets).",
			Required:            true,
			Attributes:          getCommandAttributes(),
		},
		"discover": schema.SingleNestedAttribute{
			MarkdownDescription: "The discovery config is applied to a repository. If every configured discovery tool matches, then the plugin may be used to generate manifests for Applications using the repository. If the discovery config is omitted then the plugin will not match any application but can still be invoked explicitly by specifying the plugin name in the app spec. Only one of fileName, find.glob, or find.command should be specified. If multiple are specified then only the first (in that order) is evaluated.",
			Optional:            true,
			Attributes:          getDiscoverAttributes(),
		},
		"parameters": schema.SingleNestedAttribute{
			MarkdownDescription: "The parameters config describes what parameters the UI should display for an Application. It is up to the user to actually set parameters in the Application manifest (in spec.source.plugin.parameters). The announcements only inform the \"Parameters\" tab in the App Details page of the UI.",
			Optional:            true,
			Attributes:          getParametersAttributes(),
		},
		"preserve_file_mode": schema.BoolAttribute{
			MarkdownDescription: "Whether the plugin receives repository files with original file mode. Dangerous since the repository might have executable files. Set to true only if you trust the CMP plugin authors. Set to false by default.",
			Computed:            true,
			Optional:            true,
			Default:             booldefault.StaticBool(false),
		},
	}
}

func getCommandAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"command": schema.ListAttribute{
			MarkdownDescription: "Command",
			Required:            true,
			ElementType:         types.StringType,
		},
		"args": schema.ListAttribute{
			MarkdownDescription: "Arguments of the command",
			Optional:            true,
			ElementType:         types.StringType,
		},
	}
}

func getDiscoverAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"find": schema.SingleNestedAttribute{
			MarkdownDescription: "Find config",
			Optional:            true,
			Attributes:          getFindAttributes(),
		},
		"file_name": schema.StringAttribute{
			MarkdownDescription: "A glob pattern (https://pkg.go.dev/path/filepath#Glob) that is applied to the Application's source directory. If there is a match, this plugin may be used for the Application.",
			Optional:            true,
		},
	}
}

func getFindAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"command": schema.ListAttribute{
			MarkdownDescription: "The find command runs in the repository's root directory. To match, it must exit with status code 0 and produce non-empty output to standard out.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"args": schema.ListAttribute{
			MarkdownDescription: "Arguments for the find command",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"glob": schema.StringAttribute{
			MarkdownDescription: "This does the same thing as `file_name`, but it supports double-start (nested directory) glob patterns.",
			Optional:            true,
		},
	}
}

func getParametersAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"static": schema.ListNestedAttribute{
			MarkdownDescription: "Static parameter announcements are sent to the UI for all Applications handled by this plugin. Think of the `string`, `array`, and `map` values set here as defaults. It is up to the plugin author to make sure that these default values actually reflect the plugin's behavior if the user doesn't explicitly set different values for those parameters.",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getParameterAnnouncementAttributes(),
			},
		},
		"dynamic": schema.SingleNestedAttribute{
			MarkdownDescription: "Dynamic parameter announcements are announcements specific to an Application handled by this plugin. For example, the values for a Helm chart's values.yaml file could be sent as parameter announcements.",
			Optional:            true,
			Attributes:          getDynamicAttributes(),
		},
	}
}

func getParameterAnnouncementAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			MarkdownDescription: "Parameter name",
			Optional:            true,
		},
		"title": schema.StringAttribute{
			MarkdownDescription: "Title and description of the parameter",
			Optional:            true,
		},
		"tooltip": schema.StringAttribute{
			MarkdownDescription: "Tooltip of the Parameter, will be shown when hovering over the title",
			Optional:            true,
		},
		"required": schema.BoolAttribute{
			MarkdownDescription: "Whether the Parameter is required or not. If this field is set to true, the UI will indicate to the user that they must set the value. Default to false.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"item_type": schema.StringAttribute{
			MarkdownDescription: "Item type tells the UI how to present the parameter's value (or, for arrays and maps, values). Default is `string`. Examples of other types which may be supported in the future are `boolean` or `number`. Even if the itemType is not `string`, the parameter value from the Application spec will be sent to the plugin as a string. It's up to the plugin to do the appropriate conversion.",
			Optional:            true,
		},
		"collection_type": schema.StringAttribute{
			MarkdownDescription: "Collection Type describes what type of value this parameter accepts (string, array, or map) and allows the UI to present a form to match that type. Default is `string`. This field must be present for non-string types. It will not be inferred from the presence of an `array` or `map` field.",
			Optional:            true,
		},
		"string": schema.StringAttribute{
			MarkdownDescription: "This field communicates the parameter's default value to the UI if the parameter is a `string`.",
			Optional:            true,
		},
		"array": schema.ListAttribute{
			MarkdownDescription: "This field communicates the parameter's default value to the UI if the parameter is an `array`.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"map": schema.MapAttribute{
			MarkdownDescription: "This field communicates the parameter's default value to the UI if the parameter is a `map`.",
			Optional:            true,
			ElementType:         types.StringType,
		},
	}
}

func getDynamicAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"command": schema.ListAttribute{
			MarkdownDescription: "The command will run in an Application's source directory. Standard output must be JSON matching the schema of the static parameter announcements list.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"args": schema.ListAttribute{
			MarkdownDescription: "Arguments of the command",
			Optional:            true,
			ElementType:         types.StringType,
		},
	}
}

func getAgentPermissionsRuleAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"api_groups": schema.ListAttribute{
			MarkdownDescription: "API groups of the rule.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"resources": schema.ListAttribute{
			MarkdownDescription: "Resources of the rule.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"verbs": schema.ListAttribute{
			MarkdownDescription: "Verbs of the rule.",
			Optional:            true,
			ElementType:         types.StringType,
		},
	}
}

func getCrossplaneExtensionAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"resources": schema.ListNestedAttribute{
			MarkdownDescription: "Glob patterns of the resources to match.",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getCrossplaneExtensionResourceAttributes(),
			},
		},
	}
}

func getCrossplaneExtensionResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"group": schema.StringAttribute{
			MarkdownDescription: "Glob pattern of the group to match.",
			Optional:            true,
		},
	}
}

func getAppInAnyNamespaceConfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether the app in any namespace config is enabled or not.",
			Optional:            true,
		},
	}
}

func getApplicationSetExtensionAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable ApplicationSet Extension support.",
			Optional:            true,
		},
	}
}

func getAkuityIntelligenceExtensionAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Akuity Intelligence Extension for AI-powered features",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"allowed_usernames": schema.ListAttribute{
			MarkdownDescription: "List of usernames allowed to use AI features",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"allowed_groups": schema.ListAttribute{
			MarkdownDescription: "List of groups allowed to use AI features",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"ai_support_engineer_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable AI support engineer functionality",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getClusterAddonsExtensionAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Cluster Addons Extension for managing cluster addons",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"allowed_usernames": schema.ListAttribute{
			MarkdownDescription: "List of usernames allowed to manage cluster addons",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"allowed_groups": schema.ListAttribute{
			MarkdownDescription: "List of groups allowed to manage cluster addons",
			Optional:            true,
			ElementType:         types.StringType,
		},
	}
}

func getKubeVisionConfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"cve_scan_config": schema.SingleNestedAttribute{
			MarkdownDescription: "CVE scanning configuration",
			Optional:            true,
			Attributes:          getCveScanConfigAttributes(),
		},
		"ai_config": schema.SingleNestedAttribute{
			MarkdownDescription: "AI advanced configuration like runbooks and incidents",
			Optional:            true,
			Attributes:          getAIConfigAttributes(),
		},
		"additional_attributes": schema.ListNestedAttribute{
			MarkdownDescription: "Additional attributes to include when syncing resources to KubeVision",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getAdditionalAttributeRuleAttributes(),
			},
		},
	}
}

func getAdditionalAttributeRuleAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"group": schema.StringAttribute{
			MarkdownDescription: "Kubernetes resource group",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"kind": schema.StringAttribute{
			MarkdownDescription: "Kubernetes resource kind",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"annotations": schema.ListAttribute{
			MarkdownDescription: "List of annotations to include",
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
		},
		"labels": schema.ListAttribute{
			MarkdownDescription: "List of labels to include",
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
		},
		"namespace": schema.StringAttribute{
			MarkdownDescription: "Kubernetes namespace",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getCveScanConfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"scan_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable CVE scanning",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"rescan_interval": schema.StringAttribute{
			MarkdownDescription: "CVE rescan interval",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getAIConfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"runbooks": schema.ListNestedAttribute{
			MarkdownDescription: "List of AI runbooks to use for incident resolution",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getRunbookAttributes(),
			},
		},
		"incidents": schema.SingleNestedAttribute{
			MarkdownDescription: "Incident configuration",
			Optional:            true,
			Attributes:          getIncidentsConfigAttributes(),
		},
		"argocd_slack_service": schema.StringAttribute{
			MarkdownDescription: "ArgoCD Slack service configuration",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"argocd_slack_channels": schema.ListAttribute{
			MarkdownDescription: "List of ArgoCD Slack channels",
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getRunbookAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			MarkdownDescription: "Runbook name",
			Required:            true,
		},
		"content": schema.StringAttribute{
			MarkdownDescription: "Runbook content",
			Required:            true,
		},
		"applied_to": schema.SingleNestedAttribute{
			MarkdownDescription: "Target selector for runbook application",
			Optional:            true,
			Attributes:          getTargetSelectorAttributes(),
		},
		"slack_channel_names": schema.ListAttribute{
			MarkdownDescription: "List of Slack channel names for runbook notifications",
			Optional:            true,
			ElementType:         types.StringType,
		},
	}
}

func getIncidentsConfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"triggers": schema.ListNestedAttribute{
			MarkdownDescription: "List of incident triggers",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getTargetSelectorAttributes(),
			},
		},
		"webhooks": schema.ListNestedAttribute{
			MarkdownDescription: "List of incident webhooks",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getIncidentWebhookConfigAttributes(),
			},
		},
		"grouping": schema.SingleNestedAttribute{
			MarkdownDescription: "Incident grouping configuration",
			Optional:            true,
			Attributes:          getIncidentsGroupingConfigAttributes(),
		},
	}
}

func getIncidentsGroupingConfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"k8s_namespaces": schema.ListAttribute{
			MarkdownDescription: "List of Kubernetes namespaces for incident grouping",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"argocd_application_names": schema.ListAttribute{
			MarkdownDescription: "List of ArgoCD application names for incident grouping",
			Optional:            true,
			ElementType:         types.StringType,
		},
	}
}

func getTargetSelectorAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"argocd_applications": schema.ListAttribute{
			MarkdownDescription: "List of ArgoCD applications to trigger an incident.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"k8s_namespaces": schema.ListAttribute{
			MarkdownDescription: "List of Kubernetes namespaces to trigger an incident.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"clusters": schema.ListAttribute{
			MarkdownDescription: "List of clusters to trigger an incident.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"degraded_for": schema.StringAttribute{
			MarkdownDescription: "Trigger an incident after this duration of degradation. Can be a duration string like '1h' '10m' or '10s'.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getIncidentWebhookConfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			MarkdownDescription: "Webhook name",
			Required:            true,
		},
		"description_path": schema.StringAttribute{
			MarkdownDescription: "JSON path for description field",
			Required:            true,
		},
		"cluster_path": schema.StringAttribute{
			MarkdownDescription: "JSON path for cluster field",
			Required:            true,
		},
		"k8s_namespace_path": schema.StringAttribute{
			MarkdownDescription: "JSON path for Kubernetes namespace field",
			Required:            true,
		},
		"argocd_application_name_path": schema.StringAttribute{
			MarkdownDescription: "JSON path for ArgoCD application name field",
			Required:            true,
		},
	}
}

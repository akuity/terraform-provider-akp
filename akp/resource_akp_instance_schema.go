package akp

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

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
		"argocd_notifications_cm": schema.MapAttribute{
			MarkdownDescription: "configures Argo CD notifications, and it is aligned with `argocd-notifications-cm` ConfigMap of Argo CD, for more details and examples, refer to [this documentation](https://argocd-notifications.readthedocs.io/en/stable/).",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier2.UseStateForNullUnknown(),
			},
		},
		"argocd_notifications_secret": schema.MapAttribute{
			MarkdownDescription: "contains sensitive data of Argo CD notifications, and it is aligned with `argocd-notifications-secret` Secret of Argo CD, for more details and examples, refer to [this documentation](https://argocd-notifications.readthedocs.io/en/stable/services/overview/#sensitive-data).",
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
			MarkdownDescription: "Enable Powerful AI-powered assistant Extension. It helps analyze Kubernetes resources behavior and provides suggestions about resolving issues.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
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

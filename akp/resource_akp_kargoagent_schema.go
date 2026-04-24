package akp

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	boolplanmodifier2 "github.com/akuity/terraform-provider-akp/akp/modifiers/bool"
	stringplanmodifier2 "github.com/akuity/terraform-provider-akp/akp/modifiers/string"
)

func kargoAgentSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages an AKP Kargo agent.",
		Attributes:          getAKPKargoAgentResourceAttributes(),
	}
}

func getAKPKargoAgentResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The ID of the Kargo agent",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"instance_id": schema.StringAttribute{
			MarkdownDescription: "The ID of the Kargo instance",
			Required:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"workspace": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Workspace name for the Kargo agent",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "The name of the Kargo agent",
			Required:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
			Validators: []validator.String{
				stringvalidator.LengthBetween(minClusterNameLength, maxClusterNameLength),
				stringvalidator.RegexMatches(resourceNameRegex, resourceNameRegexDescription),
			},
		},
		"namespace": schema.StringAttribute{
			MarkdownDescription: "The namespace of the Kargo agent",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"labels": schema.MapAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "Labels",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
		"annotations": schema.MapAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "Annotations",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Spec of the Kargo agent",
			Required:            true,
			Attributes:          getAKPKargoAgentSpecAttributes(),
		},
		"kube_config": schema.SingleNestedAttribute{
			MarkdownDescription: "Kubernetes connection settings. If configured, terraform will try to connect to the cluster and install the agent",
			Optional:            true,
			Attributes:          getKubeconfigAttributes(),
		},
		"remove_agent_resources_on_destroy": schema.BoolAttribute{
			MarkdownDescription: "Remove agent Kubernetes resources from the managed cluster when destroying cluster, default to `true`",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"reapply_manifests_on_update": schema.BoolAttribute{
			MarkdownDescription: "If true, re-apply generated agent manifests to the target cluster on every update when `kube_config` is provided.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
				boolplanmodifier2.SuppressProtobufDefault(),
			},
		},
	}
}

func getAKPKargoAgentSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"description": schema.StringAttribute{
			MarkdownDescription: "Description of the Kargo agent",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"data": schema.SingleNestedAttribute{
			MarkdownDescription: "Kargo agent data",
			Required:            true,
			Attributes:          getAKPKargoAgentDataAttributes(),
		},
	}
}

func getAKPKargoAgentDataAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"size": schema.StringAttribute{
			MarkdownDescription: "Cluster Size. One of `small`, `medium`, `large`. Must be omitted when `akuity_managed` is `true` because the size is managed by Akuity; use the Akuity UI or the AIMS API to change the size of an Akuity-managed agent.",
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
				boolplanmodifier2.SuppressProtobufDefault(),
			},
		},
		"target_version": schema.StringAttribute{
			MarkdownDescription: "Target version of the agent to install on your cluster",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"kustomization": schema.StringAttribute{
			MarkdownDescription: "Kustomize configuration that will be applied to generated agent installation manifests",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier2.SuppressProtobufDefault(),
			},
		},
		"remote_argocd": schema.StringAttribute{
			MarkdownDescription: "Remote Argo CD instance to connect to",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"akuity_managed": schema.BoolAttribute{
			MarkdownDescription: "This means the agent is managed by Akuity",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
				boolplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"argocd_namespace": schema.StringAttribute{
			MarkdownDescription: "Provide the namespace your Argo CD is installed in. This is only available if you self-host your Kargo agent.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"self_managed_argocd_url": schema.StringAttribute{
			MarkdownDescription: "URL of the self-managed Argo CD instance the agent connects to. This is only available if you self-host your Kargo agent.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"allowed_job_sa": schema.ListAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "List of allowed service accounts for analysis jobs created by the agent",
			Optional:            true,
		},
		"maintenance_mode": schema.BoolAttribute{
			MarkdownDescription: "Enable maintenance mode for the agent. When enabled, alerts for degraded agents are muted.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"maintenance_mode_expiry": schema.StringAttribute{
			MarkdownDescription: "Expiry time for maintenance mode in RFC3339 format. Maintenance mode will be automatically disabled after this time.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"pod_inherit_metadata": schema.BoolAttribute{
			MarkdownDescription: "Enable pod metadata inheritance. When enabled, pods inherit labels and annotations from the agent.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
				boolplanmodifier2.SuppressProtobufDefault(),
			},
		},
		"autoscaler_config": schema.SingleNestedAttribute{
			MarkdownDescription: "Autoscaler configuration for the Kargo agent.",
			Optional:            true,
			Attributes:          getKargoAutoscalerConfigAttributes(),
		},
	}
}

func getKargoAutoscalerConfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"kargo_controller": schema.SingleNestedAttribute{
			Description: "Kargo Controller auto scaling config",
			Optional:    true,
			Attributes:  getKargoControllerAutoScalingConfigAttributes(),
		},
	}
}

func getKargoControllerAutoScalingConfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"resource_minimum": schema.SingleNestedAttribute{
			Description: "Resource minimum",
			Optional:    true,
			Attributes:  getKargoResourcesAttributes(),
		},
		"resource_maximum": schema.SingleNestedAttribute{
			Description: "Resource maximum",
			Optional:    true,
			Attributes:  getKargoResourcesAttributes(),
		},
	}
}

func getKargoResourcesAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"mem": schema.StringAttribute{
			Description: "Memory",
			Optional:    true,
		},
		"cpu": schema.StringAttribute{
			Description: "CPU",
			Optional:    true,
		},
	}
}

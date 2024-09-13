package akp

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	minClusterNameLength  = 3
	maxClusterNameLength  = 50
	minInstanceNameLength = 3
	maxInstanceNameLength = 50
	minNamespaceLength    = 3
	maxNamespaceLength    = 63

	resourceNameRegex            = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
	resourceNameRegexDescription = "resource name must consist of lower case alphanumeric characters, digits or '-', and must start with an alphanumeric character, and end with an alphanumeric character or a digit"
)

func (r *AkpClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a cluster attached to an Argo CD instance.",
		Attributes:          getAKPClusterAttributes(),
	}
}

func getAKPClusterAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Cluster ID",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"instance_id": schema.StringAttribute{
			MarkdownDescription: "Argo CD instance ID",
			Required:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Cluster name",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
			Validators: []validator.String{
				stringvalidator.LengthBetween(minClusterNameLength, maxClusterNameLength),
				stringvalidator.RegexMatches(resourceNameRegex, resourceNameRegexDescription),
			},
		},
		"namespace": schema.StringAttribute{
			MarkdownDescription: "Agent installation namespace",
			Required:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier.RequiresReplace(),
			},
			Validators: []validator.String{
				stringvalidator.LengthBetween(minNamespaceLength, maxNamespaceLength),
				stringvalidator.RegexMatches(resourceNameRegex, resourceNameRegexDescription),
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
			MarkdownDescription: "Cluster spec",
			Required:            true,
			Attributes:          getClusterSpecAttributes(),
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
	}
}

func getClusterSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"description": schema.StringAttribute{
			MarkdownDescription: "Cluster description",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"namespace_scoped": schema.BoolAttribute{
			MarkdownDescription: "If the agent is namespace scoped",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
				boolplanmodifier.RequiresReplace(),
			},
		},
		"data": schema.SingleNestedAttribute{
			MarkdownDescription: "Cluster data",
			Required:            true,
			Attributes:          getClusterDataAttributes(),
		},
	}
}

func getClusterDataAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"size": schema.StringAttribute{
			MarkdownDescription: "Cluster Size. One of `small`, `medium` or `large`",
			Required:            true,
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
				boolplanmodifier.RequiresReplace(),
			},
		},
		"target_version": schema.StringAttribute{
			MarkdownDescription: "The version of the agent to install on your cluster",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
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
		"datadog_annotations_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Datadog metrics collection of Application Controller and Repo Server. Make sure that you install Datadog agent in cluster.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"eks_addon_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable this if you are installing this cluster on EKS.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"managed_cluster_config": schema.SingleNestedAttribute{
			MarkdownDescription: "The config to access managed Kubernetes cluster. By default agent is using \"in-cluster\" config.",
			Optional:            true,
			Attributes:          getManagedClusterConfigAttributes(),
		},
		"multi_cluster_k8s_dashboard_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable the KubeVision dashboards feature on the managed cluster",
			Optional:            true,
			Computed:            true,
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

func getManagedClusterConfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"secret_name": schema.StringAttribute{
			Description: "The name of the secret for the managed cluster config",
			Required:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"secret_key": schema.StringAttribute{
			Description: "The key in the secret for the managed cluster config",
			Optional:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

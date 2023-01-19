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

func (r *AkpClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Create a cluster attached to an Argo CD instance. Use `.manifests` attribute to get agent installation manifests",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Cluster ID",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"manifests": schema.StringAttribute{
				MarkdownDescription: "Agent Installation Manifests",
				Computed:            true,
				Sensitive:           true,
			},
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "Argo CD Instance ID",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Cluster Name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Cluster Description",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Agent Installation Namespace",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"namespace_scoped": schema.BoolAttribute{
				MarkdownDescription: "Agent Namespace Scoped",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
					boolplanmodifier.RequiresReplace(),
				},
			},
			"size": schema.StringAttribute{
				MarkdownDescription: "Cluster Size. One of `small`, `medium` or `large`",
				Required:            true,
			},
			"auto_upgrade_disabled": schema.BoolAttribute{
				MarkdownDescription: "Disable Agents Auto Upgrade",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"custom_image_registry_argoproj": schema.StringAttribute{
				MarkdownDescription: "Custom Registry for Argoproj Images",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"custom_image_registry_akuity": schema.StringAttribute{
				MarkdownDescription: "Custom Registry for Akuity Images",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"labels": schema.MapAttribute{
				ElementType: types.StringType,
				MarkdownDescription: "Cluster Labels",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"annotations": schema.MapAttribute{
				ElementType: types.StringType,
				MarkdownDescription: "Cluster Annotations",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"kube_config": schema.SingleNestedAttribute{
				MarkdownDescription: "Kubernetes connection setings. If configured, terraform will try to connect to the cluster and install the agent",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"host": schema.StringAttribute{
						Optional:      true,
						Description:   "The hostname (in form of URI) of Kubernetes master.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"username": schema.StringAttribute{
						Optional:      true,
						Description:   "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"password": schema.StringAttribute{
						Optional:      true,
						Sensitive:     true,
						Description:   "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"insecure": schema.BoolAttribute{
						Optional:      true,
						Description: "Whether server should be accessed without verifying the TLS certificate.",
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"client_certificate": schema.StringAttribute{
						Optional:      true,
						Description:   "PEM-encoded client certificate for TLS authentication.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"client_key": schema.StringAttribute{
						Optional:      true,
						Sensitive:     true,
						Description:   "PEM-encoded client certificate key for TLS authentication.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"cluster_ca_certificate": schema.StringAttribute{
						Optional:      true,
						Description:   "PEM-encoded root certificates bundle for TLS authentication.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"config_paths": schema.ListAttribute{
						ElementType:   types.StringType,
						Optional:      true,
						Description:   "A list of paths to kube config files. Can be set with KUBE_CONFIG_PATHS environment variable.",
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"config_path": schema.StringAttribute{
						Optional:      true,
						Description:   "Path to the kube config file.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"config_context": schema.StringAttribute{
						Optional:      true,
						Description:   "Context name to load from the kube config file.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"config_context_auth_info": schema.StringAttribute{
						Optional:      true,
						Description:   "",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"config_context_cluster": schema.StringAttribute{
						Optional:      true,
						Description:   "",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"token": schema.StringAttribute{
						Optional:      true,
						Sensitive:     true,
						Description:   "Token to authenticate an service account",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"proxy_url": schema.StringAttribute{
						Optional:      true,
						Description:   "URL to the proxy to be used for all API requests",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

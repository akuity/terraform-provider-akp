package akp

import (
	"context"

	akpstring "github.com/akuity/terraform-provider-akp/akp/planmodifiers/string"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *AkpInstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Create an Argo CD instance",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Instance ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Instance Name",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Instance Description",
				Optional:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Argo CD version. Should be equal to any [argo cd image tag](https://quay.io/repository/argoproj/argocd?tab=tags).<br>Note that `2.5.3` will result a degraded instance! Use `v2.5.3`",
				Required:            true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Instance hostname",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					akpstring.UseStateForUnknownIfNotChanged(),
				},
			},
			"admin_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable Admin Login",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_anonymous": schema.BoolAttribute{
				MarkdownDescription: "Allow Anonymous Access",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"banner": schema.SingleNestedAttribute{
				MarkdownDescription: "Argo CD Banner Configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"message": schema.StringAttribute{
						MarkdownDescription: "Banner Message",
						Required:            true,
					},
					"url": schema.StringAttribute{
						MarkdownDescription: "Banner Hyperlink URL",
						Optional:            true,
					},
					"permanent": schema.BoolAttribute{
						MarkdownDescription: "Disable hide button",
						Computed:            true,
						Optional:            true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"chat": schema.SingleNestedAttribute{
				MarkdownDescription: "Chat Configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"message": schema.StringAttribute{
						MarkdownDescription: "Alert Message",
						Required:            true,
					},
					"url": schema.StringAttribute{
						MarkdownDescription: "Alert URL",
						Optional:            true,
					},
				},
			},
			"google_analytics": schema.SingleNestedAttribute{
				MarkdownDescription: "Google Analytics Configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"tracking_id": schema.StringAttribute{
						MarkdownDescription: "Google Tracking ID",
						Required:            true,
					},
					"anonymize_users": schema.BoolAttribute{
						MarkdownDescription: "Anonymize Users",
						Required:            true,
					},
				},
			},
			"instance_label_key": schema.StringAttribute{
				MarkdownDescription: "Instance Label Key",
				Optional:            true,
			},
			"kustomize_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Enable Kustomize",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"kustomize": schema.SingleNestedAttribute{
				MarkdownDescription: "Kustomize Settings",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"build_options": schema.StringAttribute{
						MarkdownDescription: "Build options",
						Required:            true,
					},
				},
			},
			"helm_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Enable Helm",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"helm": schema.SingleNestedAttribute{
				MarkdownDescription: "Helm Configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"value_file_schemas": schema.StringAttribute{
						MarkdownDescription: "Value File Schemas",
						Required:            true,
					},
				},
			},
			"resource_settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Custom resource settings",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"inclusions": schema.StringAttribute{
						MarkdownDescription: "Inclusions",
						Optional:            true,
					},
					"exclusions": schema.StringAttribute{
						MarkdownDescription: "Exclusions",
						Optional:            true,
					},
					"compare_options": schema.StringAttribute{
						MarkdownDescription: "Compare Options",
						Optional:            true,
					},
				},
			},
			"status_badge": schema.SingleNestedAttribute{
				MarkdownDescription: "Status Badge Configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable Status Badge",
						Required:            true,
					},
					"url": schema.StringAttribute{
						MarkdownDescription: "URL",
						Optional:            true,
					},
				},
			},
			"users_session": schema.StringAttribute{
				MarkdownDescription: "Users Session Duration",
				Optional:            true,
			},
			"oidc": schema.StringAttribute{
				MarkdownDescription: "OIDC Config YAML",
				Optional:            true,
			},
			"dex": schema.StringAttribute{
				MarkdownDescription: "Dex Config YAML",
				Optional:            true,
			},
			"web_terminal": schema.SingleNestedAttribute{
				MarkdownDescription: "Web Terminal Config",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable Web Terminal",
						Required:            true,
					},
					"shells": schema.StringAttribute{
						MarkdownDescription: "Shells",
						Optional:            true,
					},
				},
			},
			"default_policy": schema.StringAttribute{
				MarkdownDescription: "Value of `policy.default` in `argocd-rbac-cm` configmap",
				Optional:            true,
			},
			"policy_csv": schema.StringAttribute{
				MarkdownDescription: "Value of `policy.csv` in `argocd-rbac-cm` configmap",
				Optional:            true,
			},
			"oidc_scopes": schema.ListAttribute{
				MarkdownDescription: "List of OIDC scopes",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"audit_extension_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable Audit Extension. Set this to `true` to install Audit Extension to Argo CD instance. Do not use `spec.extensions` for that",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"backend_ip_allow_list": schema.BoolAttribute{
				MarkdownDescription: "Apply IP Allow List to Cluster Agents",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_customization_defaults": schema.SingleNestedAttribute{
				MarkdownDescription: "Default Values For Cluster Agents",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"custom_image_registry_argoproj": schema.StringAttribute{
						MarkdownDescription: "Custom Image Registry for Argoproj images",
						Optional:            true,
					},
					"custom_image_registry_akuity": schema.StringAttribute{
						MarkdownDescription: "Custom Image Registry for Akuity images",
						Optional:            true,
					},
					"auto_upgrade_disabled": schema.BoolAttribute{
						MarkdownDescription: "Disable Agent Auto-upgrade",
						Optional:            true,
					},
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
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Extension ID",
							Required:            true,
						},
						"version": schema.StringAttribute{
							MarkdownDescription: "Extension version",
							Required:            true,
						},
					},
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
			"ip_allow_list": schema.ListNestedAttribute{
				MarkdownDescription: "IP Allow List",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							MarkdownDescription: "IP Address",
							Required:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "IP Description",
							Optional:            true,
						},
					},
				},
			},
			"repo_server_delegate": schema.SingleNestedAttribute{
				MarkdownDescription: "In case some clusters don't have network access to your private Git provider you can delegate these operations to one specific cluster.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"control_plane": schema.SingleNestedAttribute{
						MarkdownDescription: "Redundant. Always `null`",
						Computed:            true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"managed_cluster": schema.SingleNestedAttribute{
						MarkdownDescription: "Cluster",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"cluster_name": schema.StringAttribute{
								MarkdownDescription: "Cluster Name",
								Required:            true,
							},
						},
					},
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
			"secrets": schema.ListNestedAttribute{
				MarkdownDescription: "List of secrets used in SSO Configuration (OIDC or DEX config YAML)",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "Akuity API does not return secret values. Provider will try to update the secret value on every apply",
							Sensitive: true,
							Required:  true,
						},
					},
				},
			},
			// "notification_secrets": schema.ListNestedAttribute{
			// 	MarkdownDescription: "List of secrets used in Notification Settings",
			// 	Optional:            true,
			// 	NestedObject: schema.NestedAttributeObject{
			// 		Attributes: map[string]schema.Attribute{
			// 			"name": schema.StringAttribute{
			// 				Required: true,
			// 			},
			// 			"value": schema.StringAttribute{
			// 				Sensitive: true,
			// 			},
			// 		},
			// 	},
			// },
			// "image_updater_secrets": schema.ListNestedAttribute{
			// 	MarkdownDescription: "List of secrets used in Image Updater Configuration",
			// 	Optional:            true,
			// 	NestedObject: schema.NestedAttributeObject{
			// 		Attributes: map[string]schema.Attribute{
			// 			"name": schema.StringAttribute{
			// 				Required: true,
			// 			},
			// 			"value": schema.StringAttribute{
			// 				Sensitive: true,
			// 			},
			// 		},
			// 	},
			// },
		},
	}
}

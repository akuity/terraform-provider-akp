package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
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
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Argo CD version. Should be equal to any [argo cd image tag](https://quay.io/repository/argoproj/argocd?tab=tags).<br>Note that `2.5.3` will result a degraded instance! Use `v2.5.3`",
				Required:            true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Instance hostname",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"config": schema.SingleNestedAttribute{
				MarkdownDescription: "Argo CD Configuration",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"admin": schema.BoolAttribute{
						MarkdownDescription: "Enable Admin Login",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"status_badge": schema.SingleNestedAttribute{
						MarkdownDescription: "Status Badge Configuration",
						Optional:            true,
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Enable Status Badge",
								Computed:            true,
								Optional:            true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"url": schema.StringAttribute{
								MarkdownDescription: "URL",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"google_analytics": schema.SingleNestedAttribute{
						MarkdownDescription: "Google Analytics Configuration",
						Optional:            true,
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"tracking_id": schema.StringAttribute{
								MarkdownDescription: "Google Tracking ID",
								Computed:            true,
								Optional:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"anonymize_users": schema.BoolAttribute{
								MarkdownDescription: "Anonymize Users",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
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
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"message": schema.StringAttribute{
								MarkdownDescription: "Banner Message",
								Computed:            true,
								Optional:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"url": schema.StringAttribute{
								MarkdownDescription: "Banner Hyperlink URL",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"permanent": schema.BoolAttribute{
								MarkdownDescription: "Disable hide button",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"chat": schema.SingleNestedAttribute{
						MarkdownDescription: "Chat Configuration",
						Optional:            true,
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"message": schema.StringAttribute{
								MarkdownDescription: "Alert Message",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"url": schema.StringAttribute{
								MarkdownDescription: "Alert URL",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"instance_label_key": schema.StringAttribute{
						MarkdownDescription: "Instance Label Key",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"kustomize": schema.SingleNestedAttribute{
						MarkdownDescription: "Kustomize Settings",
						Optional:            true,
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Enable Kustomize",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"build_options": schema.StringAttribute{
								MarkdownDescription: "Build options",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"helm": schema.SingleNestedAttribute{
						MarkdownDescription: "Disale Agent Auto-upgrade",
						Optional:            true,
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Enable Helm",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"value_file_schemas": schema.StringAttribute{
								MarkdownDescription: "Value File Schemas",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"resource_settings": schema.SingleNestedAttribute{
						MarkdownDescription: "Custom resource settings",
						Optional:            true,
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"inclusions": schema.StringAttribute{
								MarkdownDescription: "Inclusions",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"exclusions": schema.StringAttribute{
								MarkdownDescription: "Exclusions",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"compare_options": schema.StringAttribute{
								MarkdownDescription: "Compare Options",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"users_session": schema.StringAttribute{
						MarkdownDescription: "Users Session Duration",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"oidc": schema.StringAttribute{
						MarkdownDescription: "OIDC Config YAML",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"dex": schema.StringAttribute{
						MarkdownDescription: "Dex Config YAML",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"web_terminal": schema.SingleNestedAttribute{
						MarkdownDescription: "Web Terminal Config",
						Optional:            true,
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Enable Web Terminal",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"shells": schema.StringAttribute{
								MarkdownDescription: "Shells",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
				},
			},
			"rbac_config": schema.SingleNestedAttribute{
				MarkdownDescription: "RBAC Config Map, more info [in Argo CD docs](https://argo-cd.readthedocs.io/en/stable/operator-manual/rbac/)",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"default_policy": schema.StringAttribute{
						MarkdownDescription: "Value of `policy.default` in `argocd-rbac-cm` configmap",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"policy_csv": schema.StringAttribute{
						MarkdownDescription: "Value of `policy.csv` in `argocd-rbac-cm` configmap",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"scopes": schema.ListAttribute{
						MarkdownDescription: "List of OIDC scopes",
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

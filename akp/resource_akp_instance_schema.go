package akp

import (
	"context"

	akplist "github.com/akuity/terraform-provider-akp/akp/planmodifiers/list"
	akpobject "github.com/akuity/terraform-provider-akp/akp/planmodifiers/object"
	akpstring "github.com/akuity/terraform-provider-akp/akp/planmodifiers/string"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
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
			},
			"config": schema.SingleNestedAttribute{
				MarkdownDescription: "Argo CD Configuration",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
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
						PlanModifiers: []planmodifier.Object{
							akpobject.UseStateNullForUnknown(),
						},
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
						PlanModifiers: []planmodifier.Object{
							akpobject.UseStateNullForUnknown(),
						},
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
						PlanModifiers: []planmodifier.Object{
							akpobject.UseStateNullForUnknown(),
						},
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
						PlanModifiers: []planmodifier.Object{
							akpobject.UseStateNullForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"message": schema.StringAttribute{
								MarkdownDescription: "Alert Message",
								Optional:            true,
							},
							"url": schema.StringAttribute{
								MarkdownDescription: "Alert URL",
								Optional:            true,
							},
						},
					},
					"instance_label_key": schema.StringAttribute{
						MarkdownDescription: "Instance Label Key",
						Optional:            true,
					},
					"kustomize": schema.SingleNestedAttribute{
						MarkdownDescription: "Kustomize Settings",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
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
									akpstring.UseStateNullForUnknown(),
								},
							},
						},
					},
					"helm": schema.SingleNestedAttribute{
						MarkdownDescription: "Helm Configuration",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
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
									akpstring.UseStateNullForUnknown(),
								},
							},
						},
					},
					"resource_settings": schema.SingleNestedAttribute{
						MarkdownDescription: "Custom resource settings",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Object{
							akpobject.UseStateNullForUnknown(),
						},
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
							akpstring.UseStateNullForUnknown(),
						},
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
									akpstring.UseStateNullForUnknown(),
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
					},
					"policy_csv": schema.StringAttribute{
						MarkdownDescription: "Value of `policy.csv` in `argocd-rbac-cm` configmap",
						Optional:    true,
					},
					"scopes": schema.ListAttribute{
						MarkdownDescription: "List of OIDC scopes",
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
						PlanModifiers: []planmodifier.List{
							akplist.UseStateNullForUnknown(),
						},
					},
				},
			},
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Instance Specification",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"audit_extension": schema.BoolAttribute{
						MarkdownDescription: "Enable Audit Extension",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"backend_ip_allow_list": schema.BoolAttribute{
						MarkdownDescription: "Apply IP Allow List to Cluster Agents",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"cluster_customization_defaults": schema.SingleNestedAttribute{
						MarkdownDescription: "Default Values For Cluster Agents",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Object{
							akpobject.UseStateNullForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"custom_image_registry_argoproj": schema.StringAttribute{
								MarkdownDescription: "Custom Image Registry for Argoproj images",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"custom_image_registry_akuity": schema.StringAttribute{
								MarkdownDescription: "Custom Image Registry for Akuity images",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"auto_upgrade_disabled": schema.BoolAttribute{
								MarkdownDescription: "Disable Agent Auto-upgrade",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"declarative_management": schema.BoolAttribute{
						MarkdownDescription: "Enable Declarative Management",
						Optional:            true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"extensions": schema.ListNestedAttribute{
						MarkdownDescription: "Extensions",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.List{
							akplist.UseStateNullForUnknown(),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									MarkdownDescription: "Extension ID",
									Optional:            true,
									Computed:            true,
									PlanModifiers: []planmodifier.String{
										stringplanmodifier.UseStateForUnknown(),
									},
								},
								"version": schema.StringAttribute{
									MarkdownDescription: "Extension version",
									Optional:            true,
									Computed:            true,
									PlanModifiers: []planmodifier.String{
										stringplanmodifier.UseStateForUnknown(),
									},
								},
							},
						},
					},
					"image_updater": schema.BoolAttribute{
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
						PlanModifiers: []planmodifier.List{
							akplist.UseStateNullForUnknown(),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"ip": schema.StringAttribute{
									MarkdownDescription: "IP Address",
									Optional: true,
								},
								"description": schema.StringAttribute{
									MarkdownDescription: "IP Description",
									Optional: true,
								},
							},
						},
					},
					"repo_server_delegate": schema.SingleNestedAttribute{
						MarkdownDescription: "In case some clusters don't have network access to your private Git provider you can delegate these operations to one specific cluster.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Object{
							akpobject.UseStateNullForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"control_plane": schema.SingleNestedAttribute{
								MarkdownDescription: "Redundant. Always `null`",
								Computed:    true,
								Attributes: map[string]schema.Attribute{},
								PlanModifiers: []planmodifier.Object{
									akpobject.UseStateNullForUnknown(),
								},
							},
							"managed_cluster": schema.SingleNestedAttribute{
								MarkdownDescription: "Cluster",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.Object{
									akpobject.UseStateNullForUnknown(),
								},
								Attributes: map[string]schema.Attribute{
									"cluster_name": schema.StringAttribute{
										MarkdownDescription: "Cluster Name",
										Optional: true,
										Computed: true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
								},
							},
						},
					},
					"subdomain": schema.StringAttribute{
						MarkdownDescription: "Instance Subdomain",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

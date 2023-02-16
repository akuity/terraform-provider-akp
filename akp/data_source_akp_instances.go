package akp

import (
	"context"
	"fmt"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	ctxutil "github.com/akuity/api-client-go/pkg/utils/context"
	akptypes "github.com/akuity/terraform-provider-akp/akp/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &AkpInstancesDataSource{}

func NewAkpInstancesDataSource() datasource.DataSource {
	return &AkpInstancesDataSource{}
}

// AkpInstanceDataSource defines the data source implementation.
type AkpInstancesDataSource struct {
	akpCli *AkpCli
}

type AkpInstancesDataSourceModel struct {
	Id        types.String            `tfsdk:"id"`
	Instances []*akptypes.AkpInstance `tfsdk:"instances"`
}

func (d *AkpInstancesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instances"
}

func (d *AkpInstancesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List all Argo CD instances",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"instances": schema.ListNestedAttribute{
				MarkdownDescription: "List of Argo CD instances for organization",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Instance ID",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Instance Name",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Instance Description",
							Computed:            true,
						},
						"hostname": schema.StringAttribute{
							MarkdownDescription: "Instance Hostname",
							Computed:            true,
						},
						"version": schema.StringAttribute{
							MarkdownDescription: "Argo CD Version",
							Computed:            true,
						},
						"admin_enabled": schema.BoolAttribute{
							MarkdownDescription: "Enable Admin Login",
							Computed:            true,
						},
						"status_badge": schema.SingleNestedAttribute{
							MarkdownDescription: "Status Badge Configuration",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"enabled": schema.BoolAttribute{
									MarkdownDescription: "Enable Status Badge",
									Computed:            true,
								},
								"url": schema.StringAttribute{
									MarkdownDescription: "URL",
									Computed:            true,
								},
							},
						},
						"google_analytics": schema.SingleNestedAttribute{
							MarkdownDescription: "Google Analytics Configuration",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"tracking_id": schema.StringAttribute{
									MarkdownDescription: "Google Tracking ID",
									Computed:            true,
								},
								"anonymize_users": schema.BoolAttribute{
									MarkdownDescription: "Anonymize Users",
									Computed:            true,
								},
							},
						},
						"allow_anonymous": schema.BoolAttribute{
							MarkdownDescription: "Allow Anonymous Access",
							Computed:            true,
						},
						"banner": schema.SingleNestedAttribute{
							MarkdownDescription: "Argo CD Banner Configuration",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"message": schema.StringAttribute{
									MarkdownDescription: "Banner Message",
									Computed:            true,
								},
								"url": schema.StringAttribute{
									MarkdownDescription: "Banner Hyperlink URL",
									Computed:            true,
								},
								"permanent": schema.BoolAttribute{
									MarkdownDescription: "Disable hide button",
									Computed:            true,
								},
							},
						},
						"chat": schema.SingleNestedAttribute{
							MarkdownDescription: "Chat Configuration",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"message": schema.StringAttribute{
									MarkdownDescription: "Alert Message",
									Computed:            true,
								},
								"url": schema.StringAttribute{
									MarkdownDescription: "Alert URL",
									Computed:            true,
								},
							},
						},
						"instance_label_key": schema.StringAttribute{
							MarkdownDescription: "Instance Label Key",
							Computed:            true,
						},
						"kustomize_enabled": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Enable Kustomize",
						},
						"kustomize": schema.SingleNestedAttribute{
							MarkdownDescription: "Kustomize Settings",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"build_options": schema.StringAttribute{
									MarkdownDescription: "Build options",
									Computed:            true,
								},
							},
						},
						"helm_enabled": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Enable Helm",
						},
						"helm": schema.SingleNestedAttribute{
							MarkdownDescription: "Helm Configuration",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"value_file_schemas": schema.StringAttribute{
									MarkdownDescription: "Value File Schemas",
									Computed:            true,
								},
							},
						},
						"resource_settings": schema.SingleNestedAttribute{
							MarkdownDescription: "Custom resource settings",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"inclusions": schema.StringAttribute{
									MarkdownDescription: "Inclusions",
									Computed:            true,
								},
								"exclusions": schema.StringAttribute{
									MarkdownDescription: "Exclusions",
									Computed:            true,
								},
								"compare_options": schema.StringAttribute{
									MarkdownDescription: "Compare Options",
									Computed:            true,
								},
							},
						},
						"users_session": schema.StringAttribute{
							MarkdownDescription: "Users Session Duration",
							Computed:            true,
						},
						"oidc": schema.StringAttribute{
							MarkdownDescription: "OIDC Config YAML",
							Computed:            true,
						},
						"dex": schema.StringAttribute{
							MarkdownDescription: "Dex Config YAML",
							Computed:            true,
						},
						"web_terminal": schema.SingleNestedAttribute{
							MarkdownDescription: "Web Terminal Config",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"enabled": schema.BoolAttribute{
									MarkdownDescription: "Enable Web Terminal",
									Computed:            true,
								},
								"shells": schema.StringAttribute{
									MarkdownDescription: "Shells",
									Computed:            true,
								},
							},
						},
						"default_policy": schema.StringAttribute{
							MarkdownDescription: "Value of `policy.default` in `argocd-rbac-cm` configmap",
							Computed:            true,
						},
						"policy_csv": schema.StringAttribute{
							MarkdownDescription: "Value of `policy.csv` in `argocd-rbac-cm` configmap",
							Computed:            true,
						},
						"oidc_scopes": schema.ListAttribute{
							MarkdownDescription: "List of OIDC scopes",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"audit_extension_enabled": schema.BoolAttribute{
							MarkdownDescription: "Enable Audit Extension",
							Computed:            true,
						},
						"backend_ip_allow_list": schema.BoolAttribute{
							MarkdownDescription: "Apply IP Allow List to Cluster Agents",
							Computed:            true,
						},
						"cluster_customization_defaults": schema.SingleNestedAttribute{
							MarkdownDescription: "Default Values For Cluster Agents",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"custom_image_registry_argoproj": schema.StringAttribute{
									MarkdownDescription: "Custom Image Registry for Argoproj images",
									Computed:            true,
								},
								"custom_image_registry_akuity": schema.StringAttribute{
									MarkdownDescription: "Custom Image Registry for Akuity images",
									Computed:            true,
								},
								"auto_upgrade_disabled": schema.BoolAttribute{
									MarkdownDescription: "Disable Agent Auto-upgrade",
									Computed:            true,
								},
							},
						},
						"declarative_management_enabled": schema.BoolAttribute{
							MarkdownDescription: "Enable Declarative Management",
							Computed:            true,
						},
						"extensions": schema.ListNestedAttribute{
							MarkdownDescription: "Extensions",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										MarkdownDescription: "Extension ID",
										Computed:            true,
									},
									"version": schema.StringAttribute{
										MarkdownDescription: "Extension version",
										Computed:            true,
									},
								},
							},
						},
						"image_updater_enabled": schema.BoolAttribute{
							MarkdownDescription: "Enable Image Updater",
							Computed:            true,
						},
						"ip_allow_list": schema.ListNestedAttribute{
							MarkdownDescription: "IP Allow List",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"ip": schema.StringAttribute{
										MarkdownDescription: "IP Address",
										Computed:            true,
									},
									"description": schema.StringAttribute{
										MarkdownDescription: "IP Description",
										Computed:            true,
									},
								},
							},
						},
						"repo_server_delegate": schema.SingleNestedAttribute{
							MarkdownDescription: "In case some clusters don't have network access to your private Git provider you can delegate these operations to one specific cluster.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"control_plane": schema.SingleNestedAttribute{
									MarkdownDescription: "Redundant. Always `null`",
									Computed:            true,
									Attributes:          map[string]schema.Attribute{},
								},
								"managed_cluster": schema.SingleNestedAttribute{
									MarkdownDescription: "Cluster",
									Computed:            true,
									Attributes: map[string]schema.Attribute{
										"cluster_name": schema.StringAttribute{
											MarkdownDescription: "Cluster Name",
											Computed:            true,
										},
									},
								},
							},
						},
						"subdomain": schema.StringAttribute{
							MarkdownDescription: "Instance Subdomain",
							Computed:            true,
						},
						"secrets": schema.MapNestedAttribute{
							MarkdownDescription: "Map of secrets used in SSO Configuration (OIDC or DEX config YAML)",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"value": schema.StringAttribute{
										MarkdownDescription: "Akuity API does not return secret values. In datasources this field is always null",
										Sensitive:           true,
										Computed:            true,
									},
								},
							},
						},
						"notification_secrets": schema.MapNestedAttribute{
							MarkdownDescription: "Map of secrets used in Notification Settings",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"value": schema.StringAttribute{
										Sensitive: true,
										Computed:  true,
									},
								},
							},
						},
						"image_updater": schema.SingleNestedAttribute{
							MarkdownDescription: "Image Updater Settings",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"secrets": schema.MapNestedAttribute{
									MarkdownDescription: "Map of secrets used in Image Updater Configuration",
									Computed:            true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"value": schema.StringAttribute{
												MarkdownDescription: "Api server doesn't return secret values, so this field is always null in data sources",
												Sensitive:           true,
												Computed:            true,
											},
										},
									},
								},
								"ssh_config": schema.StringAttribute{
									MarkdownDescription: "SSH Client configuration (~/.ssh/config) in Image Updater",
									Computed:            true,
								},
								"git_user": schema.StringAttribute{
									MarkdownDescription: "User name used in git commit",
									Computed:            true,
								},
								"git_email": schema.StringAttribute{
									MarkdownDescription: "User email used in git commit",
									Computed:            true,
								},
								"git_template": schema.StringAttribute{
									MarkdownDescription: "Commit Message Template for `git` write-back method. Available variables are {{\"`{{AppName}}`\"}}, {{\"`{{AppChanges}}`\"}}. [More info](https://argocd-image-updater.readthedocs.io/en/stable/basics/update-methods/#changing-the-git-commit-message)",
									Computed:            true,
								},
								"log_level": schema.StringAttribute{
									MarkdownDescription: "Log level of Image Updater Controller. One of `error`, `warn`, `info`, `debug` or `trace`",
									Computed:            true,
								},
								"registries": schema.MapNestedAttribute{
									MarkdownDescription: "Custom container registries. Not required for most public registries. [More info](https://argocd-image-updater.readthedocs.io/en/stable/configuration/registries/#configuring-custom-registries)",
									Computed:            true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"prefix": schema.StringAttribute{
												Computed: true,
											},
											"api_url": schema.StringAttribute{
												Computed: true,
											},
											"defaultns": schema.StringAttribute{
												Computed: true,
											},
											"credentials": schema.StringAttribute{
												MarkdownDescription: "Link to the configured secret. Must be in format `secret:argocd/argocd-image-updater-secret#<secret-name>`",
												Computed:            true,
											},
											"credsexpire": schema.StringAttribute{
												Computed: true,
											},
											"limit": schema.StringAttribute{
												Computed: true,
											},
											"default": schema.BoolAttribute{
												Computed: true,
											},
											"insecure": schema.BoolAttribute{
												Computed: true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *AkpInstancesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	akpCli, ok := req.ProviderData.(*AkpCli)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *AkpCli, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	d.akpCli = akpCli
}

func (d *AkpInstancesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AkpInstancesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Reading an Argo CD Instances")

	ctx = ctxutil.SetClientCredential(ctx, d.akpCli.Cred)
	apiResp, err := d.akpCli.Cli.ListInstances(ctx, &argocdv1.ListInstancesRequest{
		OrganizationId: d.akpCli.OrgId,
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Argo CD instances, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Got Argo CD instances")
	for _, instance := range apiResp.GetInstances() {
		stateInstance := &akptypes.AkpInstance{}
		err = stateInstance.Refresh(ctx, d.akpCli.Cli, d.akpCli.OrgId, instance.Id)
		if err != nil {
			resp.Diagnostics.AddError("Server Error", fmt.Sprintf("Cannot refresh instance state. %s", err))
			return
		}
		state.Instances = append(state.Instances, stateInstance)
	}

	state.Id = types.StringValue(d.akpCli.OrgId)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

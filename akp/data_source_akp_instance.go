package akp

import (
	"context"
	"fmt"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	ctxutil "github.com/akuity/api-client-go/pkg/utils/context"
	akptypes "github.com/akuity/terraform-provider-akp/akp/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &AkpInstanceDataSource{}

func NewAkpInstanceDataSource() datasource.DataSource {
	return &AkpInstanceDataSource{}
}

// AkpInstanceDataSource defines the data source implementation.
type AkpInstanceDataSource struct {
	akpCli *AkpCli
}

func (d *AkpInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (d *AkpInstanceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Find an Argo CD instance by its name",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Instance ID",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Instance Name",
				Required:            true,
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
			"config": schema.SingleNestedAttribute{
				MarkdownDescription: "Argo CD Configuration",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"admin": schema.BoolAttribute{
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
					"kustomize": schema.SingleNestedAttribute{
						MarkdownDescription: "Kustomize Settings",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Enable Kustomize",
								Computed:            true,
							},
							"build_options": schema.StringAttribute{
								MarkdownDescription: "Build options",
								Computed:            true,
							},
						},
					},
					"helm": schema.SingleNestedAttribute{
						MarkdownDescription: "Disale Agent Auto-upgrade",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Enable Helm",
								Computed:            true,
							},
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
				},
			},
			"rbac_config": schema.SingleNestedAttribute{
				MarkdownDescription: "RBAC Config Map, more info [in Argo CD docs](https://argo-cd.readthedocs.io/en/stable/operator-manual/rbac/)",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"default_policy": schema.StringAttribute{
						MarkdownDescription: "Value of `policy.default` in `argocd-rbac-cm` configmap",
						Computed:    true,
					},
					"policy_csv": schema.StringAttribute{
						MarkdownDescription: "Value of `policy.csv` in `argocd-rbac-cm` configmap",
						Computed:    true,
					},
					"scopes": schema.ListAttribute{
						MarkdownDescription: "List of OIDC scopes",
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

func (d *AkpInstanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AkpInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state *akptypes.AkpInstance
	tflog.Info(ctx, "Reading an Argo CD Instance")
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = ctxutil.SetClientCredential(ctx, d.akpCli.Cred)

	apiReq := &argocdv1.GetInstanceRequest{
		OrganizationId: d.akpCli.OrgId,
		IdType:         idv1.Type_NAME,
		Id:             state.Name.ValueString(),
	}

	tflog.Debug(ctx, fmt.Sprintf("apiReq: %s", apiReq))
	apiResp, err := d.akpCli.Cli.GetInstance(ctx, apiReq)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Argo CD instance: %s", err))
		return
	}

	tflog.Info(ctx, "Got Argo CD instance")
	resp.Diagnostics.Append(state.UpdateFrom(apiResp.GetInstance())...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

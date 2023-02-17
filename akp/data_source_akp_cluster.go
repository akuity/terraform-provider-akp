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
var _ datasource.DataSource = &AkpClustersDataSource{}

func NewAkpClusterDataSource() datasource.DataSource {
	return &AkpClusterDataSource{}
}

// AkpClustersDataSource defines the data source implementation.
type AkpClusterDataSource struct {
	akpCli *AkpCli
}

func (d *AkpClusterDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (d *AkpClusterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Find a cluster by its name and Argo CD instance ID",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Cluster ID",
				Computed:            true,
			},
			"manifests": schema.StringAttribute{
				MarkdownDescription: "Agent Installation Manifests",
				Computed:            true,
				Sensitive:           true,
			},
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "Argo CD Instance ID",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Cluster Name",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Cluster Description",
				Computed:            true,
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Agent Installation Namespace",
				Computed:            true,
			},
			"namespace_scoped": schema.BoolAttribute{
				MarkdownDescription: "Agent Namespace Scoped",
				Computed:            true,
			},
			"size": schema.StringAttribute{
				MarkdownDescription: "Cluster Size. One of `small`, `medium` or `large`",
				Computed:            true,
			},
			"auto_upgrade_disabled": schema.BoolAttribute{
				MarkdownDescription: "Disable Agents Auto Upgrade",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				ElementType: types.StringType,
				MarkdownDescription: "Cluster Labels",
				Computed:            true,
			},
			"annotations": schema.MapAttribute{
				ElementType: types.StringType,
				MarkdownDescription: "Cluster Annotations",
				Computed:            true,
			},
			"agent_version": schema.StringAttribute{
				MarkdownDescription: "Installed agent version",
				Computed:            true,
			},
		},
	}
}

func (d *AkpClusterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AkpClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state *akptypes.AkpCluster

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Reading an instance clusters")

	ctx = ctxutil.SetClientCredential(ctx, d.akpCli.Cred)
	apiReq := &argocdv1.GetInstanceClusterRequest{
		OrganizationId: d.akpCli.OrgId,
		InstanceId:     state.InstanceId.ValueString(),
		Id:             state.Name.ValueString(),
		IdType:         idv1.Type_NAME,
	}
	tflog.Debug(ctx, fmt.Sprintf("Api Request: %s", apiReq))
	apiResp, err := d.akpCli.Cli.GetInstanceCluster(ctx, apiReq)
	tflog.Debug(ctx, fmt.Sprintf("Api Response: %s", apiResp))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance clusters, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(state.UpdateCluster(apiResp.GetCluster())...)
	resp.Diagnostics.Append(state.UpdateManifests(ctx, d.akpCli.Cli, d.akpCli.OrgId)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

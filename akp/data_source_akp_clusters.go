package akp

import (
	"context"
	"fmt"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	ctxutil "github.com/akuity/api-client-go/pkg/utils/context"
	akptypes "github.com/akuity/terraform-provider-akp/akp/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &AkpClustersDataSource{}

func NewAkpClustersDataSource() datasource.DataSource {
	return &AkpClustersDataSource{}
}

// AkpClustersDataSource defines the data source implementation.
type AkpClustersDataSource struct {
	akpCli *AkpCli
}

type AkpClustersDataSourceModel struct {
	Id         types.String           `tfsdk:"id"`
	InstanceId types.String           `tfsdk:"instance_id"`
	Clusters   []*akptypes.AkpCluster `tfsdk:"clusters"`
}

func (d *AkpClustersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clusters"
}

func (d *AkpClustersDataSource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Find all clusters attached to an Argo CD instance",

		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.StringType,
				Computed:            true,
			},
			"instance_id": {
				MarkdownDescription: "Argo CD Instance ID",
				Type:                types.StringType,
				Required:            true,
			},
			"clusters": {
				MarkdownDescription: "List of clusters",
				Computed:            true,
				Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
					"id": {
						MarkdownDescription: "Cluster ID",
						Type:                types.StringType,
						Computed:            true,
					},
					"manifests": {
						MarkdownDescription: "Agent Installation Manifests",
						Type:                types.StringType,
						Computed:            true,
						Sensitive:           true,
					},
					"instance_id": {
						MarkdownDescription: "Argo CD Instance ID",
						Type:                types.StringType,
						Required:            true,
					},
					"name": {
						MarkdownDescription: "Cluster Name",
						Type:                types.StringType,
						Computed:            true,
					},
					"description": {
						MarkdownDescription: "Cluster Description",
						Type:                types.StringType,
						Computed:            true,
					},
					"namespace": {
						MarkdownDescription: "Agent Installation Namespace",
						Type:                types.StringType,
						Computed:            true,
					},
					"namespace_scoped": {
						MarkdownDescription: "Agent Namespace Scoped",
						Type:                types.BoolType,
						Computed:            true,
					},
				}),
			},
		},
	}, nil
}

func (d *AkpClustersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AkpClustersDataSource) GetManifests(ctx context.Context, instanceId string, clusterId string) (manifests string, err error) {

	tflog.Info(ctx, "Retrieving manifests...")

	apiReq := &argocdv1.GetOrganizationInstanceClusterManifestsRequest{
		OrganizationId: d.akpCli.OrgId,
		InstanceId:     instanceId,
		Id:             clusterId,
	}
	tflog.Debug(ctx, fmt.Sprintf("apiReq: %s", apiReq))
	apiResp, err := d.akpCli.Cli.GetOrganizationInstanceClusterManifests(ctx, apiReq)
	if err != nil {
		return "", err
	}

	return apiResp.GetManifests(), nil
}

func (d *AkpClustersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AkpClustersDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Reading an instance clusters")

	ctx = ctxutil.SetClientCredential(ctx, d.akpCli.Cred)
	apiResp, err := d.akpCli.Cli.ListOrganizationInstanceClusters(ctx, &argocdv1.ListOrganizationInstanceClustersRequest{
		OrganizationId: d.akpCli.OrgId,
		InstanceId:     state.InstanceId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance clusters, got error: %s", err))
		return
	}

	clusters := apiResp.GetClusters()

	for _, cluster := range clusters {
		protoCluster := &akptypes.ProtoCluster{Cluster: cluster}
		stateCluster := protoCluster.FromProto(state.InstanceId.ValueString())
		manifests, err := d.GetManifests(ctx, state.InstanceId.ValueString(), cluster.Id)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read manifests, got error: %s", err))
			return
		}
		stateCluster.Manifests = types.StringValue(manifests)
		state.Clusters = append(state.Clusters, stateCluster)
	}
	state.Id = types.StringValue(state.InstanceId.ValueString())
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

package akp

import (
	"context"
	"fmt"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
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

func (d *AkpClusterDataSource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{

		MarkdownDescription: "Find a cluster by its name and Argo CD instance ID",

		Attributes: map[string]tfsdk.Attribute{
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
				Required:            true,
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
			"size": {
				MarkdownDescription: "Cluster Size. One of `small`, `medium` or `large`",
				Type:                types.StringType,
				Computed:            true,
			},
			"auto_upgrade_disabled": {
				MarkdownDescription: "Disable Agents Auto Upgrade",
				Type:                types.BoolType,
				Computed:            true,
			},
			"custom_image_registry_argoproj": {
				MarkdownDescription: "Custom Registry for Argoproj Images",
				Type:                types.StringType,
				Computed:            true,
			},
			"custom_image_registry_akuity": {
				MarkdownDescription: "Custom Registry for Akuity Images",
				Type:                types.StringType,
				Computed:            true,
			},
			"labels": {
				MarkdownDescription: "Cluster Labels",
				Type:                types.MapType{
					ElemType: types.StringType,
				},
				Computed:            true,
			},
			"annotations": {
				MarkdownDescription: "Cluster Annotations",
				Type:                types.MapType{
					ElemType: types.StringType,
				},
				Computed:            true,
			},
		},
	}, nil
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

func (d *AkpClusterDataSource) GetManifests(ctx context.Context, instanceId string, clusterId string) (manifests string, err error) {

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

func (d *AkpClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state *akptypes.AkpCluster

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Reading an instance clusters")

	ctx = ctxutil.SetClientCredential(ctx, d.akpCli.Cred)
	apiReq := &argocdv1.GetOrganizationInstanceClusterRequest{
		OrganizationId: d.akpCli.OrgId,
		InstanceId:     state.InstanceId.ValueString(),
		Id:             state.Name.ValueString(),
		IdType:         idv1.Type_NAME,
	}
	tflog.Debug(ctx, fmt.Sprintf("Api Request: %s", apiReq))
	apiResp, err := d.akpCli.Cli.GetOrganizationInstanceCluster(ctx, apiReq)
	tflog.Debug(ctx, fmt.Sprintf("Api Response: %s", apiResp))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance clusters, got error: %s", err))
		return
	}

	cluster := apiResp.GetCluster()

	protoCluster := &akptypes.ProtoCluster{Cluster: cluster}
	state, diag := protoCluster.FromProto(state.InstanceId.ValueString())
	if diag.HasError() {
		resp.Diagnostics.Append(diag.Errors()...)
	}
	manifests, err := d.GetManifests(ctx, state.InstanceId.ValueString(), cluster.Id)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read manifests, got error: %s", err))
		return
	}
	state.Manifests = types.StringValue(manifests)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

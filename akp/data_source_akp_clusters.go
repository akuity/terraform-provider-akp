package akp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/types"
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

func (d *AkpClustersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clusters"
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

func (d *AkpClustersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data types.Clusters

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, d.akpCli.Cred.Scheme(), d.akpCli.Cred.Credential())

	apiReq := &argocdv1.ListInstanceClustersRequest{
		OrganizationId: d.akpCli.OrgId,
		InstanceId:     data.InstanceID.ValueString(),
	}
	apiResp, err := d.akpCli.Cli.ListInstanceClusters(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance clusters, got error: %s", err))
		return
	}

	clusters := apiResp.GetClusters()
	for _, cluster := range clusters {
		stateCluster := types.Cluster{
			InstanceID: data.InstanceID,
		}
		stateCluster.Update(ctx, &resp.Diagnostics, cluster)
		stateCluster.Manifests = getManifests(ctx, &resp.Diagnostics, d.akpCli.Cli, d.akpCli.OrgId, &stateCluster)
		data.Clusters = append(data.Clusters, stateCluster)
	}
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

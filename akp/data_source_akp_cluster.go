package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &AkpClusterDataSource{}

func NewAkpClusterDataSource() datasource.DataSource {
	return &AkpClusterDataSource{}
}

// AkpClusterDataSource defines the data source implementation.
type AkpClusterDataSource struct {
	BaseDataSource
}

func (d *AkpClusterDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (d *AkpClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading a Cluster Datasource")
	var data types.Cluster

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, d.akpCli.Cred.Scheme(), d.akpCli.Cred.Credential())
	if err := refreshClusterState(ctx, &resp.Diagnostics, d.akpCli.Cli, &data, d.akpCli.OrgId, nil); err != nil {
		resp.Diagnostics.AddError("Failed to refresh cluster state", err.Error())
		return
	}
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

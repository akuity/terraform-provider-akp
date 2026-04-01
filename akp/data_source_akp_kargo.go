package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &AkpKargoDataSource{}

func NewAkpKargoDataSource() datasource.DataSource {
	return &AkpKargoDataSource{}
}

// AkpKargoDataSource defines the data source implementation.
type AkpKargoDataSource struct {
	BaseDataSource
}

func (k *AkpKargoDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kargo_instance"
}

func (k *AkpKargoDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading an Instance Datasource")
	var data types.KargoInstanceDataSource
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = httpctx.SetAuthorizationHeader(ctx, k.akpCli.Cred.Scheme(), k.akpCli.Cred.Credential())

	instance := &types.KargoInstance{
		Name: data.Name,
	}
	if err := refreshKargoState(ctx, &resp.Diagnostics, k.akpCli, instance, k.akpCli.OrgId, true); err != nil {
		resp.Diagnostics.AddError("Failed to refresh kargo state", err.Error())
		return
	}
	data = types.NewKargoInstanceDataSourceModel(instance)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

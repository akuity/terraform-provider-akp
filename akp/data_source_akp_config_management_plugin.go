package akp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &AkpConfigManagementPluginDataSource{}

func NewAkpConfigManagementPluginDataSource() datasource.DataSource {
	return &AkpConfigManagementPluginDataSource{}
}

// AkpConfigManagementPluginDataSource defines the data source implementation.
type AkpConfigManagementPluginDataSource struct {
	akpCli *AkpCli
}

func (d *AkpConfigManagementPluginDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_management_plugin"
}

func (d *AkpConfigManagementPluginDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AkpConfigManagementPluginDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading a ConfigManagementPlugin Datasource")
	var data types.ConfigManagementPlugin

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, d.akpCli.Cred.Scheme(), d.akpCli.Cred.Credential())
	refreshCMPState(ctx, &resp.Diagnostics, d.akpCli.Cli, &data, d.akpCli.OrgId)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

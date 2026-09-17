package akp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

var _ datasource.DataSource = &AkpKargoAgentDataSource{}

func NewAkpKargoAgentDataSource() datasource.DataSource {
	return &AkpKargoAgentDataSource{}
}

type AkpKargoAgentDataSource struct {
	BaseDataSource
}

func (a *AkpKargoAgentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kargo_agent"
}

func (a *AkpKargoAgentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading a Kargo Agent Datasource")
	var data types.KargoAgent

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, a.akpCli.Cred.Scheme(), a.akpCli.Cred.Credential())
	if err := refreshKargoAgentState(ctx, &resp.Diagnostics, a.akpCli, &data, nil); err != nil {
		resp.Diagnostics.AddError(
			"Failed to refresh Kargo Agent state",
			fmt.Sprintf("Error: %v", err),
		)
		return
	}
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

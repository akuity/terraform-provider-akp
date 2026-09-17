package akp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
)

var _ datasource.DataSource = &AkpKargoDefaultShardAgentDataSource{}

func NewAkpKargoDefaultShardAgentDataSource() datasource.DataSource {
	return &AkpKargoDefaultShardAgentDataSource{}
}

type AkpKargoDefaultShardAgentDataSource struct {
	BaseDataSource
}

func (a *AkpKargoDefaultShardAgentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kargo_default_shard_agent"
}

func (a *AkpKargoDefaultShardAgentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading a Kargo Default Shard Agent Datasource")
	var data KargoDefaultShardAgentResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, a.akpCli.Cred.Scheme(), a.akpCli.Cred.Credential())
	instance, err := getKargoInstanceForDefaultShard(ctx, a.akpCli, data.KargoInstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Kargo instance",
			fmt.Sprintf("Error: %v", err),
		)
		return
	}

	defaultShardAgent := instance.GetSpec().GetDefaultShardAgent()
	if defaultShardAgent == "" {
		resp.Diagnostics.AddError(
			"No default shard agent configured",
			fmt.Sprintf("Kargo instance %s does not have a default shard agent configured", data.KargoInstanceID.ValueString()),
		)
		return
	}

	data.ID = data.KargoInstanceID
	data.AgentID = tftypes.StringValue(defaultShardAgent)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func (a *AkpKargoDefaultShardAgentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets information about the default shard agent for a Kargo instance",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource ID (same as kargo_instance_id)",
			},
			"kargo_instance_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the Kargo instance",
			},
			"agent_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Kargo agent set as the default shard agent",
			},
		},
	}
}

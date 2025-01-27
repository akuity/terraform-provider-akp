package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func (d *AkpKargoAgentsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets information about all Kargo agents attached to an Argo CD instance",
		Attributes:          getAKPKargoAgentsDataSourceAttributes(),
	}
}

func getAKPKargoAgentsDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"instance_id": schema.StringAttribute{
			MarkdownDescription: "Kargo instance ID",
			Computed:            true,
		},
		"id": schema.StringAttribute{
			MarkdownDescription: "Kaego instance ID",
			Computed:            true,
		},
		"agents": schema.ListNestedAttribute{
			MarkdownDescription: "List of Kargo agents",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getAKPKargoAgentDataSourceAttributes(),
			},
		},
	}
}

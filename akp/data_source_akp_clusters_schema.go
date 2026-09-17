package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func (d *AkpClustersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets information about all clusters attached to an Argo CD instance",
		Attributes:          getAKPClustersDataSourceAttributes(),
	}
}

func getAKPClustersDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"instance_id": schema.StringAttribute{
			MarkdownDescription: "Argo CD instance ID",
			Required:            true,
		},
		"id": schema.StringAttribute{
			MarkdownDescription: "Argo CD instance ID",
			Computed:            true,
		},
		"clusters": schema.ListNestedAttribute{
			MarkdownDescription: "List of clusters",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getAKPClusterDataSourceAttributes(),
			},
		},
	}
}

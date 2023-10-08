package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func (d *AkpConfigManagementPluginsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets information about all ConfigManagementPlugins attached to an Argo CD instance",
		Attributes:          getAKPConfigManagementPluginsDataSourceAttributes(),
	}
}

func getAKPConfigManagementPluginsDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"instance_id": schema.StringAttribute{
			MarkdownDescription: "Argo CD instance ID",
			Required:            true,
		},
		"plugins": schema.ListNestedAttribute{
			MarkdownDescription: "List of ConfigManagementPlugins",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getAKPConfigManagementPluginDataSourceAttributes(),
			},
		},
	}
}

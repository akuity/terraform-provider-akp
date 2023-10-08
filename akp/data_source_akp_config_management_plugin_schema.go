package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *AkpConfigManagementPluginDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets information about a config management plugin by its name and Argo CD instance ID",
		Attributes:          getAKPConfigManagementPluginDataSourceAttributes(),
	}
}

func getAKPConfigManagementPluginDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"instance_id": schema.StringAttribute{
			MarkdownDescription: "Argo CD instance ID",
			Required:            true,
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "Plugin name",
			Required:            true,
		},
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable the plugin",
			Computed:            true,
		},
		"image": schema.StringAttribute{
			MarkdownDescription: "Image to use for the plugin",
			Computed:            true,
		},
		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Plugin spec",
			Computed:            true,
			Attributes:          getPluginSpecDataSourceAttributes(),
		},
	}
}

func getPluginSpecDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"version": schema.StringAttribute{
			MarkdownDescription: "Plugin version",
			Computed:            true,
		},
		"init": schema.SingleNestedAttribute{
			MarkdownDescription: "Initialization command",
			Computed:            true,
			Attributes:          getCommandDataSourceAttributes(),
		},
		"generate": schema.SingleNestedAttribute{
			MarkdownDescription: "Generate command",
			Computed:            true,
			Attributes:          getCommandDataSourceAttributes(),
		},
		"discover": schema.SingleNestedAttribute{
			MarkdownDescription: "Discover command",
			Computed:            true,
			Attributes:          getDiscoverDataSourceAttributes(),
		},
		"parameters": schema.SingleNestedAttribute{
			MarkdownDescription: "Parameters",
			Computed:            true,
			Attributes:          getParametersDataSourceAttributes(),
		},
		"preserve_file_mode": schema.BoolAttribute{
			MarkdownDescription: "Preserve file mode",
			Computed:            true,
		},
	}
}

func getCommandDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"command": schema.ListAttribute{
			MarkdownDescription: "Command",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"args": schema.ListAttribute{
			MarkdownDescription: "Arguments",
			Computed:            true,
			ElementType:         types.StringType,
		},
	}
}

func getDiscoverDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"find": schema.SingleNestedAttribute{
			MarkdownDescription: "Find command",
			Computed:            true,
			Attributes:          getFindDataSourceAttributes(),
		},
		"file_name": schema.StringAttribute{
			MarkdownDescription: "File name",
			Computed:            true,
		},
	}
}

func getFindDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"command": schema.ListAttribute{
			MarkdownDescription: "Command",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"args": schema.ListAttribute{
			MarkdownDescription: "Arguments",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"glob": schema.StringAttribute{
			MarkdownDescription: "Glob",
			Computed:            true,
		},
	}
}

func getParametersDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"static": schema.ListNestedAttribute{
			MarkdownDescription: "Static parameters",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getParameterAnnouncementDataSourceAttributes(),
			},
		},
		"dynamic": schema.SingleNestedAttribute{
			MarkdownDescription: "Dynamic parameters",
			Computed:            true,
			Attributes:          getDynamicDataSourceAttributes(),
		},
	}
}

func getParameterAnnouncementDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			MarkdownDescription: "Name",
			Computed:            true,
		},
		"title": schema.StringAttribute{
			MarkdownDescription: "Title",
			Computed:            true,
		},
		"tooltip": schema.StringAttribute{
			MarkdownDescription: "Tooltip",
			Computed:            true,
		},
		"required": schema.BoolAttribute{
			MarkdownDescription: "Required",
			Computed:            true,
		},
		"item_type": schema.StringAttribute{
			MarkdownDescription: "Item type",
			Computed:            true,
		},
		"collection_type": schema.StringAttribute{
			MarkdownDescription: "Collection type",
			Computed:            true,
		},
		"string": schema.StringAttribute{
			MarkdownDescription: "String",
			Computed:            true,
		},
		"array": schema.ListAttribute{
			MarkdownDescription: "Array",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"map": schema.MapAttribute{
			MarkdownDescription: "Map",
			Computed:            true,
			ElementType:         types.StringType,
		},
	}
}

func getDynamicDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"command": schema.ListAttribute{
			MarkdownDescription: "Command",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"args": schema.ListAttribute{
			MarkdownDescription: "Arguments",
			Computed:            true,
			ElementType:         types.StringType,
		},
	}
}

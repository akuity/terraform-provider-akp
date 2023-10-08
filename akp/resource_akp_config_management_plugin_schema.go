package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *AkpConfigManagementPluginResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a config management plugin attached to an Argo CD instance.",
		Attributes:          getAKPConfigManagementPluginAttributes(),
	}
}

func getAKPConfigManagementPluginAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			MarkdownDescription: "Plugin name",
			Required:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"instance_id": schema.StringAttribute{
			MarkdownDescription: "Argo CD instance ID",
			Required:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable the plugin",
			Computed:            true,
			Optional:            true,
			Default:             booldefault.StaticBool(false),
		},
		"image": schema.StringAttribute{
			MarkdownDescription: "Image to use for the plugin",
			Required:            true,
		},
		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Plugin spec",
			Required:            true,
			Attributes:          getPluginSpecAttributes(),
		},
	}
}

func getPluginSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"version": schema.StringAttribute{
			MarkdownDescription: "Plugin version",
			Optional:            true,
		},
		"init": schema.SingleNestedAttribute{
			MarkdownDescription: "Initialization command",
			Optional:            true,
			Attributes:          getCommandAttributes(),
		},
		"generate": schema.SingleNestedAttribute{
			MarkdownDescription: "Generate command",
			Required:            true,
			Attributes:          getCommandAttributes(),
		},
		"discover": schema.SingleNestedAttribute{
			MarkdownDescription: "Discover command",
			Optional:            true,
			Attributes:          getDiscoverAttributes(),
		},
		"parameters": schema.SingleNestedAttribute{
			MarkdownDescription: "Parameters",
			Optional:            true,
			Attributes:          getParametersAttributes(),
		},
		"preserve_file_mode": schema.BoolAttribute{
			MarkdownDescription: "Preserve file mode",
			Computed:            true,
			Optional:            true,
			Default:             booldefault.StaticBool(false),
		},
	}
}

func getCommandAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"command": schema.ListAttribute{
			MarkdownDescription: "Command",
			Required:            true,
			ElementType:         types.StringType,
		},
		"args": schema.ListAttribute{
			MarkdownDescription: "Arguments",
			Optional:            true,
			ElementType:         types.StringType,
		},
	}
}

func getDiscoverAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"find": schema.SingleNestedAttribute{
			MarkdownDescription: "Find command",
			Optional:            true,
			Attributes:          getFindAttributes(),
		},
		"file_name": schema.StringAttribute{
			MarkdownDescription: "File name",
			Optional:            true,
		},
	}
}

func getFindAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"command": schema.ListAttribute{
			MarkdownDescription: "Command",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"args": schema.ListAttribute{
			MarkdownDescription: "Arguments",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"glob": schema.StringAttribute{
			MarkdownDescription: "Glob",
			Optional:            true,
		},
	}
}

func getParametersAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"static": schema.ListNestedAttribute{
			MarkdownDescription: "Static parameters",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getParameterAnnouncementAttributes(),
			},
		},
		"dynamic": schema.SingleNestedAttribute{
			MarkdownDescription: "Dynamic parameters",
			Optional:            true,
			Attributes:          getDynamicAttributes(),
		},
	}
}

func getParameterAnnouncementAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			MarkdownDescription: "Name",
			Optional:            true,
		},
		"title": schema.StringAttribute{
			MarkdownDescription: "Title",
			Optional:            true,
		},
		"tooltip": schema.StringAttribute{
			MarkdownDescription: "Tooltip",
			Optional:            true,
		},
		"required": schema.BoolAttribute{
			MarkdownDescription: "Required",
			Optional:            true,
		},
		"item_type": schema.StringAttribute{
			MarkdownDescription: "Item type",
			Optional:            true,
		},
		"collection_type": schema.StringAttribute{
			MarkdownDescription: "Collection type",
			Optional:            true,
		},
		"string": schema.StringAttribute{
			MarkdownDescription: "String",
			Optional:            true,
		},
		"array": schema.ListAttribute{
			MarkdownDescription: "Array",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"map": schema.MapAttribute{
			MarkdownDescription: "Map",
			Optional:            true,
			ElementType:         types.StringType,
		},
	}
}

func getDynamicAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"command": schema.ListAttribute{
			MarkdownDescription: "Command",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"args": schema.ListAttribute{
			MarkdownDescription: "Arguments",
			Optional:            true,
			ElementType:         types.StringType,
		},
	}
}

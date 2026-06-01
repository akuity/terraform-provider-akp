package akp

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func workspaceSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages an Akuity Platform workspace. Workspaces scope API keys, custom roles, members, and instances within an organization. The default workspace cannot be created or deleted via Terraform; import it if you need to manage it.",
		Attributes:          getWorkspaceAttributes(),
	}
}

func getWorkspaceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Workspace ID",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Workspace name (must be unique within the organization)",
		},
		"description": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Human-readable description for the workspace",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"create_time": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "RFC3339 timestamp of workspace creation",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"is_default": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether this is the organization's default workspace",
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

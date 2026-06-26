package akp

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func teamSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages an Akuity Platform organization team. A team groups users and can be granted custom roles; teams can also be added to workspaces via `akp_workspace_member`.",
		Attributes:          getTeamAttributes(),
	}
}

func getTeamAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Team name (must be unique within the organization). Changing this forces a new team.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"description": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Human-readable description for the team",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"custom_roles": schema.ListAttribute{
			Optional:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Organization-level custom roles granted to the team",
		},
		"create_time": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "RFC3339 timestamp of team creation",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"member_count": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Number of members in the team",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
	}
}

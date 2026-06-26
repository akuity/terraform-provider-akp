package akp

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func workspaceMemberSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Adds a member to an Akuity Platform workspace. A member is either a user (identified by `user_email`) or a team (identified by `team_name`) granted a `role` on the workspace. Exactly one of `user_email` or `team_name` must be set.",
		Attributes:          getWorkspaceMemberAttributes(),
	}
}

func getWorkspaceMemberAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Workspace membership ID",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"workspace": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Name of the workspace to add the member to",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"workspace_id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "ID of the workspace the member belongs to (resolved from `workspace`)",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"role": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Role granted to the member on the workspace. One of `member` or `admin`.",
			Validators: []validator.String{
				stringvalidator.OneOf("member", "admin"),
			},
		},
		"user_email": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Email of the user to add as a member. Mutually exclusive with `team_name`.",
			// Guard only against empty (which would satisfy ExactlyOneOf but fail
			// at apply). No full email-format check: the backend is the source of
			// truth (looks up an existing user), so the real failure is "no such
			// user", and a client-side regex would risk rejecting valid addresses.
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"team_name": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Name of the team to add as a member. Mutually exclusive with `user_email`.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
	}
}

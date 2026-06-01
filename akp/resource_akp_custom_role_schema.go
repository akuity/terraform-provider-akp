package akp

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func customRoleSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages an Akuity Platform custom role. Custom roles attach a Casbin policy that can be referenced from API keys, teams, or OIDC group mappings. A role can be scoped to the whole organization (omit `workspace`) or to a single workspace (set `workspace`). Changing `workspace` triggers replacement; `name`, `description`, and `policy` can be updated in place.",
		Attributes:          getCustomRoleAttributes(),
	}
}

func getCustomRoleAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Custom role ID",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"workspace": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Workspace name. When set, the role is scoped to this workspace; when omitted, the role is org-scoped.",
			Validators: []validator.String{
				// Reject "" explicitly — the resource logic treats null and ""
				// alike as "unset", so an empty string would silently route a
				// workspace-scoped role to the org-scoped endpoint.
				stringvalidator.LengthAtLeast(1),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Role name (must be unique within the scope)",
		},
		"description": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Human-readable description for the role",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"policy": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Casbin policy granting the role's permissions. Each non-empty, non-comment line is either a `p, sub, obj, act, resource` rule or a `g, sub, role` grouping. Server enforces additional scope checks (org policies cannot reference workspace-only objects and vice versa).",
			Validators: []validator.String{
				customRolePolicyValidator{},
			},
		},
	}
}

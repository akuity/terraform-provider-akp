package akp

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// expireInDurationRegex matches Go-style duration strings plus a `d` (days)
// suffix the server accepts. Numeric value can be int or decimal, unit is one
// of ns/us/µs/ms/s/m/h/d. Multi-unit forms like "1h30m" are accepted too.
var expireInDurationRegex = regexp.MustCompile(`^(\d+(\.\d+)?(ns|us|µs|ms|s|m|h|d))+$`)

func apiKeySchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages an Akuity Platform API key. The key may be scoped to the whole organization (omit `workspace`) or to a single workspace (set `workspace`). API keys are immutable on the server — any change to `description`, `permissions`, `expire_in_duration`, or `workspace` triggers replacement, which mints a fresh `secret`.",
		Attributes:          getApiKeyAttributes(),
	}
}

func getApiKeyAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "API key ID",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"workspace": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Workspace name. When set, the key is scoped to this workspace; when omitted, the key is org-scoped.",
			Validators: []validator.String{
				// Reject "" explicitly — the resource logic treats null and ""
				// alike as "unset", so an empty string would silently route a
				// workspace-scoped key to the org-scoped endpoint.
				stringvalidator.LengthAtLeast(1),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"description": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Human-readable description for the key",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"expire_in_duration": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Duration from creation until the key expires (e.g. `30d`, `8760h`). Omit for a non-expiring key.",
			Validators: []validator.String{
				// Reject explicit "" early — the server rejects it as
				// InvalidArgument and the error message is opaque.
				stringvalidator.LengthAtLeast(1),
				stringvalidator.RegexMatches(
					expireInDurationRegex,
					"must be a duration like `30d`, `8760h`, or `1h30m` (units: ns, us, µs, ms, s, m, h, d)",
				),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"permissions": schema.SingleNestedAttribute{
			Required:            true,
			MarkdownDescription: "Permissions granted to the key. At least one of `roles` or `custom_roles` is required.",
			PlanModifiers: []planmodifier.Object{
				// Permissions are immutable on the server; force replacement
				// at the parent level so any nested-field change (including
				// fields added later) keeps state and remote in sync.
				objectplanmodifier.RequiresReplace(),
			},
			Attributes: getApiKeyPermissionsAttributes(),
		},
		"secret": schema.StringAttribute{
			Computed:            true,
			Sensitive:           true,
			MarkdownDescription: "Generated API key secret. Only available on create or after replacement; reads from the API cannot recover it.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"organization_id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "ID of the owning organization",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"create_time": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "RFC3339 timestamp of key creation",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"expire_time": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "RFC3339 timestamp at which the key expires. Empty for non-expiring keys.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getApiKeyPermissionsAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"actions": schema.ListAttribute{
			Optional:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Action grants (uncommon; usually empty)",
			PlanModifiers: []planmodifier.List{
				listplanmodifier.RequiresReplace(),
			},
		},
		"roles": schema.ListAttribute{
			Optional:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Built-in role names. Valid org-scoped values are `owner`, `admin`, and `member`; valid workspace-scoped values are `admin` and `member`.",
			PlanModifiers: []planmodifier.List{
				listplanmodifier.RequiresReplace(),
			},
		},
		"custom_roles": schema.ListAttribute{
			Optional:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "IDs of custom roles to bind to the key",
			PlanModifiers: []planmodifier.List{
				listplanmodifier.RequiresReplace(),
			},
		},
	}
}

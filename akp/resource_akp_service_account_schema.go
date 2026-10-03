package akp

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func serviceAccountSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages an Akuity Platform service account: a non-human identity that workloads assume by exchanging a token from a trusted `akp_oidc_issuer`. The account may be scoped to the whole organization (omit `workspace`) or to a single workspace (set `workspace`), with the same role and custom role rules as `akp_api_key`. Everything but `workspace` is updated in place; changing `oidc_binding` revokes the credentials already issued to the account.",
		Attributes:          getServiceAccountAttributes(),
	}
}

func getServiceAccountAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Service account ID. Workloads pass it as `service_account_id` on the token exchange.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"workspace": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Workspace name. When set, the account is scoped to this workspace and holds workspace roles; when omitted, it is org-scoped.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"description": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Human-readable description",
			Validators: []validator.String{
				stringvalidator.LengthAtMost(255),
			},
		},
		"permissions": schema.SingleNestedAttribute{
			Required:            true,
			MarkdownDescription: "Permissions granted to the account. At least one of `roles` or `custom_roles` is required. Org-scoped accounts take `member`; workspace-scoped accounts take `admin` or `member`. The organization owner role is never allowed.",
			Attributes:          getServiceAccountPermissionsAttributes(),
		},
		"ip_allowlist": schema.ListAttribute{
			Optional:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "CIDR ranges the account's credentials may be exchanged and used from. Omit to leave it unrestricted. Not available on self-hosted installations.",
		},
		"oidc_binding": schema.SingleNestedAttribute{
			Required:            true,
			MarkdownDescription: "Which tokens may be exchanged for this account.",
			Attributes:          getServiceAccountOIDCBindingAttributes(),
		},
		"disabled": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
			MarkdownDescription: "A disabled account cannot exchange tokens and its outstanding credentials stop working immediately.",
		},
		"workspace_id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "ID of the workspace the account is scoped to. Empty for org-scoped accounts.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"create_time": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "RFC3339 timestamp of account creation",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"last_used_time": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "RFC3339 timestamp of the last request made with one of the account's credentials. Empty until first use.",
		},
	}
}

func getServiceAccountOIDCBindingAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"issuer_id": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "ID of the `akp_oidc_issuer` whose tokens are accepted",
		},
		"match": schema.MapAttribute{
			Required:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Claim name to regular expression; every entry must match the whole claim value. `sub` is required and must not match arbitrary subjects. Nested claims use dotted paths, e.g. `kubernetes.io.namespace`.",
			Validators: []validator.Map{
				mapvalidator.KeysAre(stringvalidator.LengthAtLeast(1)),
			},
		},
	}
}

// getServiceAccountPermissionsAttributes is the API key permissions block
// with the roles a service account may hold: the same as an API key of the
// same scope, minus the organization owner role.
func getServiceAccountPermissionsAttributes() map[string]schema.Attribute {
	attrs := getApiKeyPermissionsAttributes()
	attrs["roles"] = schema.ListAttribute{
		Optional:            true,
		ElementType:         types.StringType,
		MarkdownDescription: "Built-in role names. Org-scoped accounts take `member`; workspace-scoped accounts take `admin` or `member`. `owner` is never allowed.",
		Validators: []validator.List{
			listvalidator.ValueStringsAre(stringvalidator.NoneOf("owner")),
		},
	}
	return attrs
}

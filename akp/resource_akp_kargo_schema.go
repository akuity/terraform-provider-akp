package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *AkpKargoInstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AKP Kargo instance.",
		Attributes:          getAKPKargoInstanceAttributes(),
	}
}

func getAKPKargoInstanceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Kargo Instance ID",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Kargo Instance name",
			Validators: []validator.String{
				stringvalidator.LengthBetween(minInstanceNameLength, maxInstanceNameLength),
				stringvalidator.RegexMatches(resourceNameRegex, resourceNameRegexDescription),
			},
		},
		"kargo": schema.SingleNestedAttribute{
			Required:            true,
			MarkdownDescription: "Kargo instance configuration",
			Attributes:          getKargoAttributes(),
		},
	}
}

func getKargoAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Kargo instance spec",
			Required:            true,
			Attributes:          getKargoSpecAttributes(),
		},
	}
}

func getKargoSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"description": schema.StringAttribute{
			MarkdownDescription: "Description of the Kargo instance",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"version": schema.StringAttribute{
			MarkdownDescription: "Version of the Kargo instance",
			Required:            true,
		},
		"kargo_instance_spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Kargo instance spec",
			Required:            true,
			Attributes:          getKargoSpecInstanceAttributes(),
		},
	}
}

func getKargoSpecInstanceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"backend_ip_allow_list_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether IP allow list is enabled for the backend",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"ip_allow_list": schema.ListNestedAttribute{
			MarkdownDescription: "List of allowed IPs",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: getKargoIPAllowListEntryAttributes(),
			},
		},
		"agent_customization_defaults": schema.SingleNestedAttribute{
			MarkdownDescription: "Default agent customization settings",
			Optional:            true,
			Attributes:          getKargoAgentCustomizationAttributes(),
		},
		"default_shard_agent": schema.StringAttribute{
			MarkdownDescription: "Default shard agent",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"global_credentials_ns": schema.ListAttribute{
			MarkdownDescription: "List of global credentials namespaces",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"global_service_account_ns": schema.ListAttribute{
			MarkdownDescription: "List of global service account namespaces",
			Optional:            true,
			ElementType:         types.StringType,
		},
	}
}

func getKargoIPAllowListEntryAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ip": schema.StringAttribute{
			MarkdownDescription: "IP Address",
			Required:            true,
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "Description",
			Optional:            true,
		},
	}
}

func getKargoAgentCustomizationAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"auto_upgrade_disabled": schema.BoolAttribute{
			MarkdownDescription: "Whether auto upgrade is disabled",
			Optional:            true,
			Default:             booldefault.StaticBool(false),
			Computed:            true,
		},
		"kustomization": schema.StringAttribute{
			MarkdownDescription: "Kustomization that will be applied to the Kargo agent to generate agent installation manifests",
			Optional:            true,
		},
	}
}

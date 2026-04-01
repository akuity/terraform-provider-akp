package akp

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	boolplanmodifier2 "github.com/akuity/terraform-provider-akp/akp/modifiers/bool"
	objectplanmodifier2 "github.com/akuity/terraform-provider-akp/akp/modifiers/object"
	stringplanmodifier2 "github.com/akuity/terraform-provider-akp/akp/modifiers/string"
)

func kargoInstanceSchema() schema.Schema {
	return schema.Schema{
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
		"kargo_cm": schema.MapAttribute{
			MarkdownDescription: "ConfigMap to configure system account accesses. The usage can be found in the examples/resources/akp_kargo_instance/resource.tf",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
		"kargo_secret": schema.MapAttribute{
			MarkdownDescription: "Secret to configure system account accesses. The usage can be found in the examples/resources/akp_kargo_instance/resource.tf",
			ElementType:         types.StringType,
			Optional:            true,
			Sensitive:           true,
		},
		"workspace": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Workspace name for the Kargo instance",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier2.SuppressProtobufDefault(),
			},
		},
		"kargo_resources": schema.MapAttribute{
			MarkdownDescription: "Map of Kargo custom resources to be managed alongside the Kargo instance. Currently supported resources are: `Project`, `ClusterPromotionTask`, `Stage`, `Warehouse`, `AnalysisTemplate`, `PromotionTask`(with Groups: `kargo.akuity.io`); `Secret`(only with `kargo.akuity.io/cred-type` label); `ConfigMap`; `Role`, `RoleBinding`, `ServiceAccount`(`rbac.kargo.akuity.io/managed=\"true\"` annotation required)",
			Optional:            true,
			ElementType:         types.StringType,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
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
		"fqdn": schema.StringAttribute{
			MarkdownDescription: "Fully qualified domain name",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier2.SuppressProtobufDefault(),
			},
		},
		"subdomain": schema.StringAttribute{
			MarkdownDescription: "Subdomain of the Kargo instance",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"oidc_config": schema.SingleNestedAttribute{
			MarkdownDescription: "OIDC configuration",
			Optional:            true,
			Computed:            true,
			Attributes:          getKargoOidcConfigAttributes(),
			PlanModifiers: []planmodifier.Object{
				objectplanmodifier2.UseStateForNullUnknown(),
			},
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
			Computed:            true,
			Attributes:          getKargoAgentCustomizationAttributes(),
			PlanModifiers: []planmodifier.Object{
				objectplanmodifier2.UseStateForNullUnknown(),
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
		"akuity_intelligence": schema.SingleNestedAttribute{
			MarkdownDescription: "Akuity Intelligence configuration for AI-powered features",
			Optional:            true,
			Computed:            true,
			Attributes:          getKargoAkuityIntelligenceAttributes(),
			PlanModifiers: []planmodifier.Object{
				objectplanmodifier2.UseStateForNullUnknown(),
			},
		},
		"gc_config": schema.SingleNestedAttribute{
			MarkdownDescription: "Garbage collector configuration",
			Optional:            true,
			Computed:            true,
			Attributes:          getGarbageCollectorConfigAttributes(),
			PlanModifiers: []planmodifier.Object{
				objectplanmodifier2.UseStateForNullUnknown(),
			},
		},
		"promo_controller_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether Kargo Promotion Controller is enabled for this instance",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
				boolplanmodifier2.SuppressProtobufDefault(),
			},
		},
		"argocd_ui": schema.SingleNestedAttribute{
			MarkdownDescription: "Controls behavior of Argo CD user interface in the Kargo instance",
			Optional:            true,
			Attributes:          getKargoArgoCDUIConfigAttributes(),
		},
	}
}

func getKargoArgoCDUIConfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"idp_groups_mapping": schema.BoolAttribute{
			MarkdownDescription: "When enabled, group claims from the user's Kargo IDP token are mapped to Argo CD to authorize edit operations",
			Optional:            true,
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

func getKargoOidcConfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether OIDC is enabled",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"dex_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether DEX is enabled",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"dex_config": schema.StringAttribute{
			MarkdownDescription: "DEX configuration",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier2.SuppressProtobufDefault(),
			},
		},
		"dex_config_secret": schema.MapAttribute{
			MarkdownDescription: "DEX configuration secret",
			Optional:            true,
			Sensitive:           true,
			ElementType:         types.StringType,
		},
		"issuer_url": schema.StringAttribute{
			MarkdownDescription: "Issuer URL",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier2.SuppressProtobufDefault(),
			},
		},
		"client_id": schema.StringAttribute{
			MarkdownDescription: "Client ID",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier2.SuppressProtobufDefault(),
			},
		},
		"cli_client_id": schema.StringAttribute{
			MarkdownDescription: "CLI Client ID",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier2.SuppressProtobufDefault(),
			},
		},
		"admin_account": schema.SingleNestedAttribute{
			MarkdownDescription: "Admin account",
			Optional:            true,
			Computed:            true,
			Attributes:          getKargoPredefinedAccountAttributes(),
			PlanModifiers: []planmodifier.Object{
				objectplanmodifier2.UseStateForNullUnknown(),
			},
		},
		"viewer_account": schema.SingleNestedAttribute{
			MarkdownDescription: "Viewer account",
			Optional:            true,
			Computed:            true,
			Attributes:          getKargoPredefinedAccountAttributes(),
			PlanModifiers: []planmodifier.Object{
				objectplanmodifier2.UseStateForNullUnknown(),
			},
		},
		"additional_scopes": schema.ListAttribute{
			MarkdownDescription: "Additional scopes",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"user_account": schema.SingleNestedAttribute{
			MarkdownDescription: "User account",
			Optional:            true,
			Computed:            true,
			Attributes:          getKargoPredefinedAccountAttributes(),
			PlanModifiers: []planmodifier.Object{
				objectplanmodifier2.UseStateForNullUnknown(),
			},
		},
		"project_creator_account": schema.SingleNestedAttribute{
			MarkdownDescription: "Project creator account",
			Optional:            true,
			Computed:            true,
			Attributes:          getKargoPredefinedAccountAttributes(),
			PlanModifiers: []planmodifier.Object{
				objectplanmodifier2.UseStateForNullUnknown(),
			},
		},
	}
}

func getKargoPredefinedAccountAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"claims": schema.MapNestedAttribute{
			MarkdownDescription: "Claims",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"values": schema.SetAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
					},
				},
			},
		},
	}
}

func getGarbageCollectorConfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"max_retained_freight": schema.Int64Attribute{
			MarkdownDescription: "Maximum number of freight objects to retain",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"max_retained_promotions": schema.Int64Attribute{
			MarkdownDescription: "Maximum number of promotion objects to retain",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"min_freight_deletion_age": schema.Int64Attribute{
			MarkdownDescription: "Minimum age in seconds before freight objects can be deleted",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"min_promotion_deletion_age": schema.Int64Attribute{
			MarkdownDescription: "Minimum age in seconds before promotion objects can be deleted",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getKargoAkuityIntelligenceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ai_support_engineer_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable AI support engineer functionality",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Akuity Intelligence for AI-powered features",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"allowed_usernames": schema.ListAttribute{
			MarkdownDescription: "List of usernames allowed to use AI features",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"allowed_groups": schema.ListAttribute{
			MarkdownDescription: "List of groups allowed to use AI features",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"model_version": schema.StringAttribute{
			MarkdownDescription: "AI model version to use",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier2.SuppressProtobufDefault(),
			},
		},
	}
}

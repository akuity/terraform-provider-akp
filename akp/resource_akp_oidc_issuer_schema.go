package akp

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	oidcKeySourceDiscovery  = "discovery"
	oidcKeySourceJWKSURI    = "jwks_uri"
	oidcKeySourceStaticJWKS = "static_jwks"

	oidcDefaultTokenTTLSeconds = 3600
)

func oidcIssuerSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages a trusted OIDC issuer for the organization. Service accounts bind to an issuer so workloads it identifies (a GitHub Actions workflow, a Kubernetes service account, an Entra application) can exchange their tokens for Akuity credentials. `name` is immutable; every other attribute is updated in place. Changing the issuer URL or its keys revokes every credential issued to the accounts bound to it.",
		Attributes:          getOIDCIssuerAttributes(),
	}
}

func getOIDCIssuerAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Issuer ID",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Unique name within the organization. Immutable: changing it replaces the issuer, which breaks the bindings of its service accounts.",
			Validators: []validator.String{
				stringvalidator.LengthBetween(1, 255),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"issuer_url": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The `iss` claim tokens carry, e.g. `https://token.actions.githubusercontent.com`. Must be https unless the server allows insecure issuer URLs.",
		},
		"key_source": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Where signing keys come from: `discovery` reads them from the issuer's `.well-known/openid-configuration`, `jwks_uri` fetches the URL in `jwks_uri`, `static_jwks` uses the document in `static_jwks`.",
			Validators: []validator.String{
				stringvalidator.OneOf(oidcKeySourceDiscovery, oidcKeySourceJWKSURI, oidcKeySourceStaticJWKS),
			},
		},
		"jwks_uri": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "JWKS URL. Required when `key_source` is `jwks_uri`, must be unset otherwise.",
		},
		"static_jwks": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "JWKS document as a JSON string, e.g. via `jsonencode` or `file`. Required when `key_source` is `static_jwks`, must be unset otherwise. Only RSA (2048 bits or more) and EC signing keys are used.",
		},
		"audiences": schema.ListAttribute{
			Optional:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Audiences accepted on subject tokens. Omit to accept only the server-assigned `default_audience`; set for providers that cannot mint a custom audience (for example Microsoft Entra).",
		},
		"token_ttl_seconds": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(oidcDefaultTokenTTLSeconds),
			MarkdownDescription: "Lifetime of the credentials minted for this issuer's service accounts, between 60 and 86400 seconds. Defaults to 3600.",
			Validators: []validator.Int64{
				int64validator.Between(60, 86400),
			},
		},
		"default_audience": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The audience the server assigns to this organization; subject tokens must carry it unless `audiences` is set.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"keys_updated_time": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "RFC3339 timestamp of the last change to the issuer's key configuration. Empty until the first change.",
		},
		"create_time": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "RFC3339 timestamp of issuer creation",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

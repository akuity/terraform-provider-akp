package types

import (
	"context"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type AkpArgoCDConfig struct {
	AdminEnabled          types.Bool   `tfsdk:"admin"`
	AllowAnonymousUser    types.Bool   `tfsdk:"allow_anonymous"`
	Banner                types.Object `tfsdk:"banner"`
	Chat                  types.Object `tfsdk:"chat"`
	DexConfig             types.String `tfsdk:"dex"`
	GoogleAnalytics       types.Object `tfsdk:"google_analytics"`
	HelmSettings          types.Object `tfsdk:"helm"`
	InstanceLabelKey      types.String `tfsdk:"instance_label_key"`
	KustomizeSettings     types.Object `tfsdk:"kustomize"`
	OidcConfig            types.String `tfsdk:"oidc"`
	ResourceSettings      types.Object `tfsdk:"resource_settings"`
	StatusBadge           types.Object `tfsdk:"status_badge"`
	UsersSessionDuration  types.String `tfsdk:"users_session"`
	WebTerminal           types.Object `tfsdk:"web_terminal"`
}

var (
	configMapAttrTypes = map[string]attr.Type{
		"admin":            types.BoolType,
		"allow_anonymous": types.BoolType,
		"banner": types.ObjectType{
			AttrTypes: bannerAttrTypes,
		},
		"chat": types.ObjectType{
			AttrTypes: chatAttrTypes,
		},
		"dex": types.StringType,
		"google_analytics": types.ObjectType{
			AttrTypes: googleAnalyticsAttrTypes,
		},
		"helm": types.ObjectType{
			AttrTypes: helmSettingsAttrTypes,
		},
		"instance_label_key": types.StringType,
		"kustomize": types.ObjectType{
			AttrTypes: kustomizeSettingsAttrTypes,
		},
		"oidc": types.StringType,
		"resource_settings": types.ObjectType{
			AttrTypes: resourceSettingsAttrTypes,
		},
		"status_badge":     types.ObjectType{
			AttrTypes: statusBadgeAttrTypes,
		},
		"users_session": types.StringType,
		"web_terminal": types.ObjectType{
			AttrTypes: webTerminalAttrTypes,
		},
	}
)

func (x *AkpArgoCDConfig) UpdateObject(p *argocdv1.ArgoCDConfigMap) diag.Diagnostics {
	diags := diag.Diagnostics{}
	d := diag.Diagnostics{}
	x.AdminEnabled = types.BoolValue(p.GetAdminEnabled())
	x.AllowAnonymousUser = types.BoolValue(p.GetAllowAnonymousUser())
	banner := &AkpArgoCDBanner{}
	diags.Append(banner.UpdateObject(p.GetBanner())...)
	x.Banner, d  = types.ObjectValueFrom(context.Background(), bannerAttrTypes, &banner)
	diags.Append(d...)
	chat := &AkpArgoCDChat{}
	diags.Append(chat.UpdateObject(p.GetChat())...)
	x.Chat, d  = types.ObjectValueFrom(context.Background(), chatAttrTypes, &chat)
	diags.Append(d...)
	x.DexConfig = types.StringValue(p.GetDexConfig())
	googleAnalytics := &AkpArgoCDGoogleAnalytics{}
	diags.Append(googleAnalytics.UpdateObject(p.GetGoogleAnalytics())...)
	x.GoogleAnalytics, d = types.ObjectValueFrom(context.Background(), googleAnalyticsAttrTypes, &googleAnalytics)
	diags.Append(d...)
	helmSettings := &AkpArgoCDHelmSettings{}
	diags.Append(helmSettings.UpdateObject(p.GetHelmSettings())...)
	x.HelmSettings, d = types.ObjectValueFrom(context.Background(), helmSettingsAttrTypes, &helmSettings)
	diags.Append(d...)
	x.InstanceLabelKey = types.StringValue(p.GetInstanceLabelKey())
	kustomizeSettings := &AkpArgoCDKustomizeSettings{}
	diags.Append(kustomizeSettings.UpdateObject(p.GetKustomizeSettings())...)
	x.KustomizeSettings, d = types.ObjectValueFrom(context.Background(), kustomizeSettingsAttrTypes, &kustomizeSettings)
	diags.Append(d...)
	x.OidcConfig = types.StringValue(p.GetOidcConfig())
	resourceSettings := &AkpArgoCDResourceSettings{}
	diags.Append(resourceSettings.UpdateObject(p.GetResourceSettings())...)
	x.ResourceSettings, d = types.ObjectValueFrom(context.Background(), resourceSettingsAttrTypes, &resourceSettings)
	diags.Append(d...)
	statusBadge := &AkpArgoCDStatusBadge{}
	diags.Append(statusBadge.UpdateObject(p.GetStatusBadge())...)
	x.StatusBadge, d = types.ObjectValueFrom(context.Background(), statusBadgeAttrTypes, &statusBadge)
	diags.Append(d...)
	x.UsersSessionDuration = types.StringValue(p.GetUsersSessionDuration())
	webTerminal := &AkpArgoCDWebTerminal{}
	diags.Append(webTerminal.UpdateObject(p.GetWebTerminal())...)
	x.WebTerminal, d = types.ObjectValueFrom(context.Background(), webTerminalAttrTypes, &webTerminal)
	diags.Append(d...)
	return diags
}

func (x *AkpArgoCDConfig) As(target *argocdv1.ArgoCDConfigMap) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.AdminEnabled = x.AdminEnabled.ValueBool()
	target.AllowAnonymousUser = x.AllowAnonymousUser.ValueBool()
	var banner AkpArgoCDBanner
	diags.Append(x.Banner.As(context.Background(),&banner, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty: true,
		UnhandledUnknownAsEmpty: true,
	})...)
	diags.Append(banner.As(target.Banner)...)
	var chat AkpArgoCDChat
	diags.Append(x.Chat.As(context.Background(),&chat, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty: true,
		UnhandledUnknownAsEmpty: true,
	})...)
	diags.Append(chat.As(target.Chat)...)
	target.DexConfig = x.DexConfig.ValueString()
	var googleAnalytics AkpArgoCDGoogleAnalytics
	diags.Append(x.GoogleAnalytics.As(context.Background(),&googleAnalytics, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty: true,
		UnhandledUnknownAsEmpty: true,
	})...)
	diags.Append(googleAnalytics.As(target.GoogleAnalytics)...)
	var helmSettings AkpArgoCDHelmSettings
	diags.Append(x.HelmSettings.As(context.Background(),&helmSettings, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty: true,
		UnhandledUnknownAsEmpty: true,
	})...)
	diags.Append(helmSettings.As(target.HelmSettings)...)
	target.InstanceLabelKey = x.InstanceLabelKey.ValueString()
	var kustomizeSettings AkpArgoCDKustomizeSettings
	diags.Append(x.KustomizeSettings.As(context.Background(),&kustomizeSettings, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty: true,
		UnhandledUnknownAsEmpty: true,
	})...)
	diags.Append(kustomizeSettings.As(target.KustomizeSettings)...)
	target.OidcConfig = x.OidcConfig.ValueString()
	var resourceSettings AkpArgoCDResourceSettings
	diags.Append(x.ResourceSettings.As(context.Background(),&resourceSettings, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty: true,
		UnhandledUnknownAsEmpty: true,
	})...)
	diags.Append(resourceSettings.As(target.ResourceSettings)...)
	var statusBadge AkpArgoCDStatusBadge
	diags.Append(x.StatusBadge.As(context.Background(),&statusBadge, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty: true,
		UnhandledUnknownAsEmpty: true,
	})...)
	diags.Append(statusBadge.As(target.StatusBadge)...)
	target.UsersSessionDuration = x.UsersSessionDuration.ValueString()
	var webTerminal AkpArgoCDWebTerminal
	diags.Append(x.WebTerminal.As(context.Background(),&webTerminal, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty: true,
		UnhandledUnknownAsEmpty: true,
	})...)
	diags.Append(webTerminal.As(target.WebTerminal)...)
	return diags
}

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
	AdminEnabled         types.Bool   `tfsdk:"admin"`
	AllowAnonymousUser   types.Bool   `tfsdk:"allow_anonymous"`
	Banner               types.Object `tfsdk:"banner"`
	Chat                 types.Object `tfsdk:"chat"`
	DexConfig            types.String `tfsdk:"dex"`
	GoogleAnalytics      types.Object `tfsdk:"google_analytics"`
	HelmSettings         types.Object `tfsdk:"helm"`
	InstanceLabelKey     types.String `tfsdk:"instance_label_key"`
	KustomizeSettings    types.Object `tfsdk:"kustomize"`
	OidcConfig           types.String `tfsdk:"oidc"`
	ResourceSettings     types.Object `tfsdk:"resource_settings"`
	StatusBadge          types.Object `tfsdk:"status_badge"`
	UsersSessionDuration types.String `tfsdk:"users_session"`
	WebTerminal          types.Object `tfsdk:"web_terminal"`
}

var (
	configMapAttrTypes = map[string]attr.Type{
		"admin":           types.BoolType,
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
		"status_badge": types.ObjectType{
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
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ArgoCDConfigMap is <nil>")
		return diags
	}
	x.AdminEnabled = types.BoolValue(p.GetAdminEnabled())
	x.AllowAnonymousUser = types.BoolValue(p.GetAllowAnonymousUser())
	if p.Banner == nil {
		x.Banner = types.ObjectNull(bannerAttrTypes)
	} else {
		banner := &AkpArgoCDBanner{}
		diags.Append(banner.UpdateObject(p.Banner)...)
		x.Banner, d = types.ObjectValueFrom(context.Background(), bannerAttrTypes, &banner)
		diags.Append(d...)
	}

	if p.Chat == nil {
		x.Chat = types.ObjectNull(chatAttrTypes)
	} else {
		chat := &AkpArgoCDChat{}
		diags.Append(chat.UpdateObject(p.Chat)...)
		x.Chat, d = types.ObjectValueFrom(context.Background(), chatAttrTypes, &chat)
		diags.Append(d...)
	}

	x.DexConfig = types.StringValue(p.GetDexConfig())

	if p.GoogleAnalytics == nil {
		x.GoogleAnalytics = types.ObjectNull(googleAnalyticsAttrTypes)
	} else {
		googleAnalytics := &AkpArgoCDGoogleAnalytics{}
		diags.Append(googleAnalytics.UpdateObject(p.GoogleAnalytics)...)
		x.GoogleAnalytics, d = types.ObjectValueFrom(context.Background(), googleAnalyticsAttrTypes, &googleAnalytics)
		diags.Append(d...)
	}

	if p.HelmSettings == nil {
		x.HelmSettings = types.ObjectNull(helmSettingsAttrTypes)
	} else {
		helmSettings := &AkpArgoCDHelmSettings{}
		diags.Append(helmSettings.UpdateObject(p.HelmSettings)...)
		x.HelmSettings, d = types.ObjectValueFrom(context.Background(), helmSettingsAttrTypes, &helmSettings)
		diags.Append(d...)
	}

	x.InstanceLabelKey = types.StringValue(p.GetInstanceLabelKey())

	if p.KustomizeSettings == nil {
		x.KustomizeSettings = types.ObjectNull(kustomizeSettingsAttrTypes)
	} else {
		kustomizeSettings := &AkpArgoCDKustomizeSettings{}
		diags.Append(kustomizeSettings.UpdateObject(p.KustomizeSettings)...)
		x.KustomizeSettings, d = types.ObjectValueFrom(context.Background(), kustomizeSettingsAttrTypes, &kustomizeSettings)
		diags.Append(d...)
	}

	x.OidcConfig = types.StringValue(p.GetOidcConfig())

	if p.ResourceSettings == nil {
		x.ResourceSettings = types.ObjectNull(resourceSettingsAttrTypes)
	} else {
		resourceSettings := &AkpArgoCDResourceSettings{}
		diags.Append(resourceSettings.UpdateObject(p.ResourceSettings)...)
		x.ResourceSettings, d = types.ObjectValueFrom(context.Background(), resourceSettingsAttrTypes, &resourceSettings)
		diags.Append(d...)
	}

	if p.StatusBadge == nil {
		x.StatusBadge = types.ObjectNull(statusBadgeAttrTypes)
	} else {
		statusBadge := &AkpArgoCDStatusBadge{}
		diags.Append(statusBadge.UpdateObject(p.StatusBadge)...)
		x.StatusBadge, d = types.ObjectValueFrom(context.Background(), statusBadgeAttrTypes, &statusBadge)
		diags.Append(d...)
	}
	x.UsersSessionDuration = types.StringValue(p.GetUsersSessionDuration())

	if p.WebTerminal == nil {
		x.WebTerminal = types.ObjectNull(webTerminalAttrTypes)
	} else {
		webTerminal := &AkpArgoCDWebTerminal{}
		diags.Append(webTerminal.UpdateObject(p.WebTerminal)...)
		x.WebTerminal, d = types.ObjectValueFrom(context.Background(), webTerminalAttrTypes, &webTerminal)
		diags.Append(d...)
	}
	return diags
}

func (x *AkpArgoCDConfig) As(target *argocdv1.ArgoCDConfigMap) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.AdminEnabled = x.AdminEnabled.ValueBool()
	target.AllowAnonymousUser = x.AllowAnonymousUser.ValueBool()

	if x.Banner.IsNull() {
		target.Banner = nil
	} else if !x.Banner.IsUnknown() {
		banner := AkpArgoCDBanner{}
		targetBanner := argocdv1.ArgoCDBannerConfig{}
		diags.Append(x.Banner.As(context.Background(), &banner, basetypes.ObjectAsOptions{})...)
		diags.Append(banner.As(&targetBanner)...)
		target.Banner = &targetBanner
	}

	if x.Chat.IsNull() {
		target.Chat = nil
	} else if !x.Chat.IsUnknown() {
		chat := AkpArgoCDChat{}
		targetChat := argocdv1.ArgoCDAlertConfig{}
		diags.Append(x.Chat.As(context.Background(), &chat, basetypes.ObjectAsOptions{})...)
		diags.Append(chat.As(&targetChat)...)
		target.Chat = &targetChat
	}

	target.DexConfig = x.DexConfig.ValueString()

	if x.GoogleAnalytics.IsNull() {
		target.GoogleAnalytics = nil
	} else if !x.GoogleAnalytics.IsUnknown() {
		googleAnalytics := AkpArgoCDGoogleAnalytics{}
		targetGoogleAnalytics := argocdv1.ArgoCDGoogleAnalyticsConfig{}
		diags.Append(x.GoogleAnalytics.As(context.Background(), &googleAnalytics, basetypes.ObjectAsOptions{})...)
		diags.Append(googleAnalytics.As(&targetGoogleAnalytics)...)
		target.GoogleAnalytics = &targetGoogleAnalytics
	}

	if x.HelmSettings.IsNull() {
		target.HelmSettings = nil
	} else if !x.HelmSettings.IsUnknown() {
		helmSettings := AkpArgoCDHelmSettings{}
		targetHelmSettings := argocdv1.ArgoCDHelmSettings{}
		diags.Append(x.HelmSettings.As(context.Background(), &helmSettings, basetypes.ObjectAsOptions{})...)
		diags.Append(helmSettings.As(&targetHelmSettings)...)
		target.HelmSettings = &targetHelmSettings
	}

	target.InstanceLabelKey = x.InstanceLabelKey.ValueString()

	if x.KustomizeSettings.IsNull() {
		target.KustomizeSettings = nil
	} else if !x.KustomizeSettings.IsUnknown() {
		kustomizeSettings := AkpArgoCDKustomizeSettings{}
		targetKustomizeSettings := argocdv1.ArgoCDKustomizeSettings{}
		diags.Append(x.KustomizeSettings.As(context.Background(), &kustomizeSettings, basetypes.ObjectAsOptions{})...)
		diags.Append(kustomizeSettings.As(&targetKustomizeSettings)...)
		target.KustomizeSettings = &targetKustomizeSettings
	}

	target.OidcConfig = x.OidcConfig.ValueString()

	if x.ResourceSettings.IsNull() {
		target.ResourceSettings = nil
	} else if !x.ResourceSettings.IsUnknown() {
		resourceSettings := AkpArgoCDResourceSettings{}
		targetResourceSettings := argocdv1.ArgoCDResourceSettings{}
		diags.Append(x.ResourceSettings.As(context.Background(), &resourceSettings, basetypes.ObjectAsOptions{})...)
		diags.Append(resourceSettings.As(&targetResourceSettings)...)
		target.ResourceSettings = &targetResourceSettings
	}

	if x.StatusBadge.IsNull() {
		target.StatusBadge = nil
	} else if !x.StatusBadge.IsUnknown() {
		statusBadge := AkpArgoCDStatusBadge{}
		targetStatusBadge := argocdv1.ArgoCDStatusBadgeConfig{}
		diags.Append(x.StatusBadge.As(context.Background(), &statusBadge, basetypes.ObjectAsOptions{})...)
		diags.Append(statusBadge.As(&targetStatusBadge)...)
		target.StatusBadge = &targetStatusBadge
	}

	target.UsersSessionDuration = x.UsersSessionDuration.ValueString()

	if x.WebTerminal.IsNull() {
		target.WebTerminal = nil
	} else if !x.WebTerminal.IsUnknown() {
		webTerminal := AkpArgoCDWebTerminal{}
		targetWebTerminal := argocdv1.ArgoCDWebTerminalConfig{}
		diags.Append(x.WebTerminal.As(context.Background(), &webTerminal, basetypes.ObjectAsOptions{})...)
		diags.Append(webTerminal.As(&targetWebTerminal)...)
		target.WebTerminal = &targetWebTerminal
	}

	return diags
}

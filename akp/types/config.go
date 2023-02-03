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

func MergeConfig(state *AkpArgoCDConfig, plan *AkpArgoCDConfig) (*AkpArgoCDConfig, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpArgoCDConfig{}

	if plan.AdminEnabled.IsUnknown() {
		res.AdminEnabled = state.AdminEnabled
	} else if plan.AdminEnabled.IsNull() {
		res.AdminEnabled = types.BoolNull()
	} else {
		res.AdminEnabled = plan.AdminEnabled
	}

	if plan.AllowAnonymousUser.IsUnknown() {
		res.AllowAnonymousUser = state.AllowAnonymousUser
	} else if plan.AllowAnonymousUser.IsNull() {
		res.AllowAnonymousUser = types.BoolNull()
	} else {
		res.AllowAnonymousUser = plan.AllowAnonymousUser
	}

	if plan.Banner.IsUnknown() {
		res.Banner = state.Banner
	} else if plan.Banner.IsNull() {
		res.Banner = types.ObjectNull(bannerAttrTypes)
	} else {
		var stateBanner, planBanner AkpArgoCDBanner
		diags.Append(state.Banner.As(context.Background(), &stateBanner, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.Banner.As(context.Background(), &planBanner, basetypes.ObjectAsOptions{})...)
		resBanner, d := MergeBanner(&stateBanner, &planBanner)
		diags.Append(d...)
		res.Banner, d = types.ObjectValueFrom(context.Background(), bannerAttrTypes, resBanner)
		diags.Append(d...)
	}

	if plan.Chat.IsUnknown() {
		res.Chat = state.Chat
	} else if plan.Chat.IsNull() {
		res.Chat = types.ObjectNull(chatAttrTypes)
	} else {
		var stateChat, planChat AkpArgoCDChat
		diags.Append(state.Chat.As(context.Background(), &stateChat, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.Chat.As(context.Background(), &planChat, basetypes.ObjectAsOptions{})...)
		resChat, d := MergeChat(&stateChat, &planChat)
		diags.Append(d...)
		res.Chat, d = types.ObjectValueFrom(context.Background(), chatAttrTypes, resChat)
		diags.Append(d...)
	}

	if plan.DexConfig.IsUnknown() {
		res.DexConfig = state.DexConfig
	} else if plan.DexConfig.IsNull() {
		res.DexConfig = types.StringNull()
	} else {
		res.DexConfig = plan.DexConfig
	}

	if plan.GoogleAnalytics.IsUnknown() {
		res.GoogleAnalytics = state.GoogleAnalytics
	} else if plan.GoogleAnalytics.IsNull() {
		res.GoogleAnalytics = types.ObjectNull(googleAnalyticsAttrTypes)
	} else {
		var stateGoogleAnalytics, planGoogleAnalytics AkpArgoCDGoogleAnalytics
		diags.Append(state.GoogleAnalytics.As(context.Background(), &stateGoogleAnalytics, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.GoogleAnalytics.As(context.Background(), &planGoogleAnalytics, basetypes.ObjectAsOptions{})...)
		resGoogleAnalytics, d := MergeGoogleAnalytics(&stateGoogleAnalytics, &planGoogleAnalytics)
		diags.Append(d...)
		res.GoogleAnalytics, d = types.ObjectValueFrom(context.Background(), googleAnalyticsAttrTypes, resGoogleAnalytics)
		diags.Append(d...)
	}

	if plan.HelmSettings.IsUnknown() {
		res.HelmSettings = state.HelmSettings
	} else if plan.HelmSettings.IsNull() {
		res.HelmSettings = types.ObjectNull(helmSettingsAttrTypes)
	} else {
		var stateHelmSettings, planHelmSettings AkpArgoCDHelmSettings
		diags.Append(state.HelmSettings.As(context.Background(), &stateHelmSettings, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.HelmSettings.As(context.Background(), &planHelmSettings, basetypes.ObjectAsOptions{})...)
		resHelmSettings, d := MergeHelmSettings(&stateHelmSettings, &planHelmSettings)
		diags.Append(d...)
		res.HelmSettings, d = types.ObjectValueFrom(context.Background(), helmSettingsAttrTypes, resHelmSettings)
		diags.Append(d...)
	}

	if plan.InstanceLabelKey.IsUnknown() {
		res.InstanceLabelKey = state.InstanceLabelKey
	} else if plan.InstanceLabelKey.IsNull() {
		res.InstanceLabelKey = types.StringNull()
	} else {
		res.InstanceLabelKey = plan.InstanceLabelKey
	}

	if plan.KustomizeSettings.IsUnknown() {
		res.KustomizeSettings = state.KustomizeSettings
	} else if plan.KustomizeSettings.IsNull() {
		res.KustomizeSettings = types.ObjectNull(kustomizeSettingsAttrTypes)
	} else {
		var stateKustomizeSettings, planKustomizeSettings AkpArgoCDKustomizeSettings
		diags.Append(state.KustomizeSettings.As(context.Background(), &stateKustomizeSettings, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.KustomizeSettings.As(context.Background(), &planKustomizeSettings, basetypes.ObjectAsOptions{})...)
		resKustomizeSettings, d := MergeKustomizeSettings(&stateKustomizeSettings, &planKustomizeSettings)
		diags.Append(d...)
		res.KustomizeSettings, d = types.ObjectValueFrom(context.Background(), kustomizeSettingsAttrTypes, resKustomizeSettings)
		diags.Append(d...)
	}

	if plan.OidcConfig.IsUnknown() {
		res.OidcConfig = state.OidcConfig
	} else if plan.OidcConfig.IsNull() {
		res.OidcConfig = types.StringNull()
	} else {
		res.OidcConfig = plan.OidcConfig
	}

	if plan.ResourceSettings.IsUnknown() {
		res.ResourceSettings = state.ResourceSettings
	} else if plan.ResourceSettings.IsNull() {
		res.ResourceSettings = types.ObjectNull(resourceSettingsAttrTypes)
	} else {
		var stateResourceSettings, planResourceSettings AkpArgoCDResourceSettings
		diags.Append(state.ResourceSettings.As(context.Background(), &stateResourceSettings, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.ResourceSettings.As(context.Background(), &planResourceSettings, basetypes.ObjectAsOptions{})...)
		resResourceSettings, d := MergeResourceSettings(&stateResourceSettings, &planResourceSettings)
		diags.Append(d...)
		res.ResourceSettings, d = types.ObjectValueFrom(context.Background(), resourceSettingsAttrTypes, resResourceSettings)
		diags.Append(d...)
	}

	if plan.StatusBadge.IsUnknown() {
		res.StatusBadge = state.StatusBadge
	} else if plan.StatusBadge.IsNull() {
		res.StatusBadge = types.ObjectNull(statusBadgeAttrTypes)
	} else {
		var stateStatusBadge, planStatusBadge AkpArgoCDStatusBadge
		diags.Append(state.StatusBadge.As(context.Background(), &stateStatusBadge, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.StatusBadge.As(context.Background(), &planStatusBadge, basetypes.ObjectAsOptions{})...)
		resStatusBadge, d := MergeStatusBadge(&stateStatusBadge, &planStatusBadge)
		diags.Append(d...)
		res.StatusBadge, d = types.ObjectValueFrom(context.Background(), statusBadgeAttrTypes, resStatusBadge)
		diags.Append(d...)
	}

	if plan.UsersSessionDuration.IsUnknown() {
		res.UsersSessionDuration = state.UsersSessionDuration
	} else if plan.UsersSessionDuration.IsNull() {
		res.UsersSessionDuration = types.StringNull()
	} else {
		res.UsersSessionDuration = plan.UsersSessionDuration
	}

	if plan.WebTerminal.IsUnknown() {
		res.WebTerminal = state.WebTerminal
	} else if plan.WebTerminal.IsNull() {
		res.WebTerminal = types.ObjectNull(webTerminalAttrTypes)
	} else {
		var stateWebTerminal, planWebTerminal AkpArgoCDWebTerminal
		diags.Append(state.WebTerminal.As(context.Background(), &stateWebTerminal, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.WebTerminal.As(context.Background(), &planWebTerminal, basetypes.ObjectAsOptions{})...)
		resWebTerminal, d := MergeWebTerminal(&stateWebTerminal, &planWebTerminal)
		diags.Append(d...)
		res.WebTerminal, d = types.ObjectValueFrom(context.Background(), webTerminalAttrTypes, resWebTerminal)
		diags.Append(d...)
	}

	return res, diags
}

func (x *AkpArgoCDConfig) UpdateObject(p *argocdv1.ArgoCDConfigMap) diag.Diagnostics {
	diags := diag.Diagnostics{}
	d := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ArgoCDConfigMap is <nil>")
		return diags
	}
	x.AdminEnabled = types.BoolValue(p.GetAdminEnabled())
	x.AllowAnonymousUser = types.BoolValue(p.GetAllowAnonymousUser())
	if p.Banner == nil || p.Banner.String() == "" {
		x.Banner = types.ObjectNull(bannerAttrTypes)
	} else {
		banner := &AkpArgoCDBanner{}
		diags.Append(banner.UpdateObject(p.Banner)...)
		x.Banner, d = types.ObjectValueFrom(context.Background(), bannerAttrTypes, &banner)
		diags.Append(d...)
	}

	if p.Chat == nil || p.Chat.String() == "" {
		x.Chat = types.ObjectNull(chatAttrTypes)
	} else {
		chat := &AkpArgoCDChat{}
		diags.Append(chat.UpdateObject(p.Chat)...)
		x.Chat, d = types.ObjectValueFrom(context.Background(), chatAttrTypes, &chat)
		diags.Append(d...)
	}

	if p.DexConfig == "" {
		x.DexConfig = types.StringNull()
	} else {
		x.DexConfig = types.StringValue(p.DexConfig)
	}

	if p.GoogleAnalytics == nil || p.GoogleAnalytics.String() == "" {
		x.GoogleAnalytics = types.ObjectNull(googleAnalyticsAttrTypes)
	} else {
		googleAnalytics := &AkpArgoCDGoogleAnalytics{}
		diags.Append(googleAnalytics.UpdateObject(p.GoogleAnalytics)...)
		x.GoogleAnalytics, d = types.ObjectValueFrom(context.Background(), googleAnalyticsAttrTypes, &googleAnalytics)
		diags.Append(d...)
	}

	if p.HelmSettings == nil || p.HelmSettings.String() == "" {
		x.HelmSettings = types.ObjectNull(helmSettingsAttrTypes)
	} else {
		helmSettings := &AkpArgoCDHelmSettings{}
		diags.Append(helmSettings.UpdateObject(p.HelmSettings)...)
		x.HelmSettings, d = types.ObjectValueFrom(context.Background(), helmSettingsAttrTypes, &helmSettings)
		diags.Append(d...)
	}

	if p.InstanceLabelKey == "" {
		x.InstanceLabelKey = types.StringNull()
	} else {
		x.InstanceLabelKey = types.StringValue(p.InstanceLabelKey)
	}

	if p.KustomizeSettings == nil || p.KustomizeSettings.String() == "" {
		x.KustomizeSettings = types.ObjectNull(kustomizeSettingsAttrTypes)
	} else {
		kustomizeSettings := &AkpArgoCDKustomizeSettings{}
		diags.Append(kustomizeSettings.UpdateObject(p.KustomizeSettings)...)
		x.KustomizeSettings, d = types.ObjectValueFrom(context.Background(), kustomizeSettingsAttrTypes, &kustomizeSettings)
		diags.Append(d...)
	}

	if p.OidcConfig == "" {
		x.OidcConfig = types.StringNull()
	} else {
		x.OidcConfig = types.StringValue(p.OidcConfig)
	}

	if p.ResourceSettings == nil || p.ResourceSettings.String() == "" {
		x.ResourceSettings = types.ObjectNull(resourceSettingsAttrTypes)
	} else {
		resourceSettings := &AkpArgoCDResourceSettings{}
		diags.Append(resourceSettings.UpdateObject(p.ResourceSettings)...)
		x.ResourceSettings, d = types.ObjectValueFrom(context.Background(), resourceSettingsAttrTypes, &resourceSettings)
		diags.Append(d...)
	}

	if p.StatusBadge == nil || p.StatusBadge.String() == "" {
		x.StatusBadge = types.ObjectNull(statusBadgeAttrTypes)
	} else {
		statusBadge := &AkpArgoCDStatusBadge{}
		diags.Append(statusBadge.UpdateObject(p.StatusBadge)...)
		x.StatusBadge, d = types.ObjectValueFrom(context.Background(), statusBadgeAttrTypes, &statusBadge)
		diags.Append(d...)
	}

	if p.UsersSessionDuration == "" {
		x.UsersSessionDuration = types.StringNull()
	} else {
		x.UsersSessionDuration = types.StringValue(p.UsersSessionDuration)
	}

	if p.WebTerminal == nil || p.WebTerminal.String() == "" {
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
		if target.Banner != nil {
			diags.Append(banner.UpdateObject(target.Banner)...)
		}
		targetBanner := argocdv1.ArgoCDBannerConfig{}
		diags.Append(x.Banner.As(context.Background(), &banner, basetypes.ObjectAsOptions{})...)
		diags.Append(banner.As(&targetBanner)...)
		target.Banner = &targetBanner
	}

	if x.Chat.IsNull() {
		target.Chat = nil
	} else if !x.Chat.IsUnknown() {
		chat := AkpArgoCDChat{}
		if target.Chat != nil {
			diags.Append(chat.UpdateObject(target.Chat)...)
		}
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
		if target.GoogleAnalytics != nil {
			diags.Append(googleAnalytics.UpdateObject(target.GoogleAnalytics)...)
		}
		targetGoogleAnalytics := argocdv1.ArgoCDGoogleAnalyticsConfig{}
		diags.Append(x.GoogleAnalytics.As(context.Background(), &googleAnalytics, basetypes.ObjectAsOptions{})...)
		diags.Append(googleAnalytics.As(&targetGoogleAnalytics)...)
		target.GoogleAnalytics = &targetGoogleAnalytics
	}

	if x.HelmSettings.IsNull() {
		target.HelmSettings = nil
	} else if !x.HelmSettings.IsUnknown() {
		helmSettings := AkpArgoCDHelmSettings{}
		if target.HelmSettings != nil {
			diags.Append(helmSettings.UpdateObject(target.HelmSettings)...)
		}
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
		if target.KustomizeSettings != nil {
			diags.Append(kustomizeSettings.UpdateObject(target.KustomizeSettings)...)
		}
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
		if target.ResourceSettings != nil {
			diags.Append(resourceSettings.UpdateObject(target.ResourceSettings)...)
		}
		targetResourceSettings := argocdv1.ArgoCDResourceSettings{}
		diags.Append(x.ResourceSettings.As(context.Background(), &resourceSettings, basetypes.ObjectAsOptions{})...)
		diags.Append(resourceSettings.As(&targetResourceSettings)...)
		target.ResourceSettings = &targetResourceSettings
	}

	if x.StatusBadge.IsNull() {
		target.StatusBadge = nil
	} else if !x.StatusBadge.IsUnknown() {
		statusBadge := AkpArgoCDStatusBadge{}
		if target.StatusBadge != nil {
			diags.Append(statusBadge.UpdateObject(target.StatusBadge)...)
		}
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
		if target.WebTerminal != nil {
			diags.Append(webTerminal.UpdateObject(target.WebTerminal)...)
		}
		targetWebTerminal := argocdv1.ArgoCDWebTerminalConfig{}
		diags.Append(x.WebTerminal.As(context.Background(), &webTerminal, basetypes.ObjectAsOptions{})...)
		diags.Append(webTerminal.As(&targetWebTerminal)...)
		target.WebTerminal = &targetWebTerminal
	}

	return diags
}

package types

import (
	"context"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type AkpInstance struct {
	Id                    types.String `tfsdk:"id"`                             // computed
	Name                  types.String `tfsdk:"name"`                           // required
	Version               types.String `tfsdk:"version"`                        // required
	Description           types.String `tfsdk:"description"`                    // optional
	Hostname              types.String `tfsdk:"hostname"`                       // computed
	AuditExtension        types.Bool   `tfsdk:"audit_extension_enabled"`        // optional computed
	BackendIpAllowList    types.Bool   `tfsdk:"backend_ip_allow_list"`          // optional computed
	ClusterCustomization  types.Object `tfsdk:"cluster_customization_defaults"` // optional
	DeclarativeManagement types.Bool   `tfsdk:"declarative_management_enabled"` // optional computed
	Extensions            types.List   `tfsdk:"extensions"`                     // optional
	ImageUpdater          types.Bool   `tfsdk:"image_updater_enabled"`          // optional computed
	IpAllowList           types.List   `tfsdk:"ip_allow_list"`                  // optional
	RepoServerDelegate    types.Object `tfsdk:"repo_server_delegate"`           // optional
	Subdomain             types.String `tfsdk:"subdomain"`                      // optional computed
	AdminEnabled          types.Bool   `tfsdk:"admin_enabled"`                  // optional computed
	AllowAnonymousUser    types.Bool   `tfsdk:"allow_anonymous"`                // optional computed
	Banner                types.Object `tfsdk:"banner"`                         // optional
	Chat                  types.Object `tfsdk:"chat"`                           // optional
	DexConfig             types.String `tfsdk:"dex"`                            // optional
	GoogleAnalytics       types.Object `tfsdk:"google_analytics"`               // optional
	HelmEnabled           types.Bool   `tfsdk:"helm_enabled"`                   // optional computed
	HelmSettings          types.Object `tfsdk:"helm"`                           // optional
	InstanceLabelKey      types.String `tfsdk:"instance_label_key"`             // optional
	KustomizeEnabled      types.Bool   `tfsdk:"kustomize_enabled"`              // optional computed
	KustomizeSettings     types.Object `tfsdk:"kustomize"`                      // optional
	OidcConfig            types.String `tfsdk:"oidc"`                           // optional
	ResourceSettings      types.Object `tfsdk:"resource_settings"`              // optional
	StatusBadge           types.Object `tfsdk:"status_badge"`                   // optional
	UsersSessionDuration  types.String `tfsdk:"users_session"`                  // optional
	WebTerminal           types.Object `tfsdk:"web_terminal"`                   // optional
	DefaultPolicy         types.String `tfsdk:"default_policy"`                 // optional
	PolicyCsv             types.String `tfsdk:"policy_csv"`                     // optional
	OidcScopes            types.List   `tfsdk:"oidc_scopes"`                    // optional
}

func MergeInstance(state *AkpInstance, plan *AkpInstance) (*AkpInstance, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpInstance{
		Id:       state.Id,
		Name:     state.Name,
		Version:  state.Version,
		Hostname: state.Hostname,
	}
	if plan.Description.IsUnknown() {
		res.Description = state.Description
	} else {
		res.Description = plan.Description
	}

	// ------- Config -------
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

	if plan.HelmEnabled.IsUnknown() {
		res.HelmEnabled = state.HelmEnabled
	} else if plan.HelmEnabled.IsNull() {
		res.HelmEnabled = types.BoolNull()
	} else {
		res.HelmEnabled = plan.HelmEnabled
	}

	if plan.HelmSettings.IsUnknown() {
		res.HelmSettings = state.HelmSettings
	} else if plan.HelmSettings.IsNull() {
		res.HelmSettings = types.ObjectNull(HelmSettingsAttrTypes)
	} else {
		var stateHelmSettings, planHelmSettings AkpArgoCDHelmSettings
		diags.Append(state.HelmSettings.As(context.Background(), &stateHelmSettings, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.HelmSettings.As(context.Background(), &planHelmSettings, basetypes.ObjectAsOptions{})...)
		resHelmSettings, d := MergeHelmSettings(&stateHelmSettings, &planHelmSettings)
		diags.Append(d...)
		res.HelmSettings, d = types.ObjectValueFrom(context.Background(), HelmSettingsAttrTypes, resHelmSettings)
		diags.Append(d...)
	}

	if plan.InstanceLabelKey.IsUnknown() {
		res.InstanceLabelKey = state.InstanceLabelKey
	} else if plan.InstanceLabelKey.IsNull() {
		res.InstanceLabelKey = types.StringNull()
	} else {
		res.InstanceLabelKey = plan.InstanceLabelKey
	}

	if plan.KustomizeEnabled.IsUnknown() {
		res.KustomizeEnabled = state.KustomizeEnabled
	} else if plan.KustomizeEnabled.IsNull() {
		res.KustomizeEnabled = types.BoolNull()
	} else {
		res.KustomizeEnabled = plan.KustomizeEnabled
	}

	if plan.KustomizeSettings.IsUnknown() {
		res.KustomizeSettings = state.KustomizeSettings
	} else if plan.KustomizeSettings.IsNull() {
		res.KustomizeSettings = types.ObjectNull(KustomizeSettingsAttrTypes)
	} else {
		var stateKustomizeSettings, planKustomizeSettings AkpArgoCDKustomizeSettings
		diags.Append(state.KustomizeSettings.As(context.Background(), &stateKustomizeSettings, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.KustomizeSettings.As(context.Background(), &planKustomizeSettings, basetypes.ObjectAsOptions{})...)
		resKustomizeSettings, d := MergeKustomizeSettings(&stateKustomizeSettings, &planKustomizeSettings)
		diags.Append(d...)
		res.KustomizeSettings, d = types.ObjectValueFrom(context.Background(), KustomizeSettingsAttrTypes, resKustomizeSettings)
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
	// ----- RBAC ------
	if plan.DefaultPolicy.IsUnknown() {
		res.DefaultPolicy = state.DefaultPolicy
	} else if plan.DefaultPolicy.IsNull() {
		res.DefaultPolicy = types.StringNull()
	} else {
		res.DefaultPolicy = plan.DefaultPolicy
	}

	if plan.PolicyCsv.IsUnknown() {
		res.PolicyCsv = state.PolicyCsv
	} else if plan.PolicyCsv.IsNull() {
		res.PolicyCsv = types.StringNull()
	} else {
		res.PolicyCsv = plan.PolicyCsv
	}

	if plan.OidcScopes.IsUnknown() {
		res.OidcScopes = state.OidcScopes
	} else if plan.OidcScopes.IsNull() {
		res.OidcScopes = types.ListNull(types.StringType)
	} else {
		res.OidcScopes = plan.OidcScopes
	}

	if plan.AuditExtension.IsUnknown() {
		res.AuditExtension = state.AuditExtension
	} else if plan.AuditExtension.IsNull() {
		res.AuditExtension = types.BoolNull()
	} else {
		res.AuditExtension = plan.AuditExtension
	}

	if plan.BackendIpAllowList.IsUnknown() {
		res.BackendIpAllowList = state.BackendIpAllowList
	} else if plan.BackendIpAllowList.IsNull() {
		res.BackendIpAllowList = types.BoolNull()
	} else {
		res.BackendIpAllowList = plan.BackendIpAllowList
	}

	// ------- Spec -------
	if plan.ClusterCustomization.IsUnknown() {
		res.ClusterCustomization = state.ClusterCustomization
	} else if plan.ClusterCustomization.IsNull() {
		res.ClusterCustomization = types.ObjectNull(clusterCustomizationAttrTypes)
	} else {
		var stateClusterCustomization, planClusterCustomization AkpClusterCustomization
		diags.Append(state.ClusterCustomization.As(context.Background(), &stateClusterCustomization, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.ClusterCustomization.As(context.Background(), &planClusterCustomization, basetypes.ObjectAsOptions{})...)
		resClusterCustomization, d := MergeClusterCustomization(&stateClusterCustomization, &planClusterCustomization)
		diags.Append(d...)
		res.ClusterCustomization, d = types.ObjectValueFrom(context.Background(), clusterCustomizationAttrTypes, resClusterCustomization)
		diags.Append(d...)
	}

	if plan.DeclarativeManagement.IsUnknown() {
		res.DeclarativeManagement = state.DeclarativeManagement
	} else if plan.DeclarativeManagement.IsNull() {
		res.DeclarativeManagement = types.BoolNull()
	} else {
		res.DeclarativeManagement = plan.DeclarativeManagement
	}

	if plan.Extensions.IsUnknown() {
		res.Extensions = state.Extensions
	} else if plan.Extensions.IsNull() {
		res.Extensions = types.ListNull(types.ObjectType{AttrTypes: extensionInstallEntryAttrTypes})
	} else {
		res.Extensions = plan.Extensions
	}

	if plan.ImageUpdater.IsUnknown() {
		res.ImageUpdater = state.ImageUpdater
	} else if plan.ImageUpdater.IsNull() {
		res.ImageUpdater = types.BoolNull()
	} else {
		res.ImageUpdater = plan.ImageUpdater
	}

	if plan.IpAllowList.IsUnknown() {
		res.IpAllowList = state.IpAllowList
	} else if plan.IpAllowList.IsNull() {
		res.IpAllowList = types.ListNull(types.ObjectType{AttrTypes: iPAllowListEntryAttrTypes})
	} else {
		res.IpAllowList = plan.IpAllowList
	}

	if plan.RepoServerDelegate.IsUnknown() {
		res.RepoServerDelegate = state.RepoServerDelegate
	} else if plan.RepoServerDelegate.IsNull() {
		res.RepoServerDelegate = types.ObjectNull(repoServerDelegateAttrTypes)
	} else {
		var stateRepoServerDelegate, planRepoServerDelegate AkpRepoServerDelegate
		diags.Append(state.RepoServerDelegate.As(context.Background(), &stateRepoServerDelegate, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty: true,
		})...)
		diags.Append(plan.RepoServerDelegate.As(context.Background(), &planRepoServerDelegate, basetypes.ObjectAsOptions{})...)
		resRepoServerDelegate, d := MergeRepoServerDelegate(&stateRepoServerDelegate, &planRepoServerDelegate)
		diags.Append(d...)
		res.RepoServerDelegate, d = types.ObjectValueFrom(context.Background(), repoServerDelegateAttrTypes, resRepoServerDelegate)
		diags.Append(d...)
	}

	if plan.Subdomain.IsUnknown() {
		res.Subdomain = state.Subdomain
	} else if plan.Subdomain.IsNull() {
		res.Subdomain = types.StringNull()
	} else {
		res.Subdomain = plan.Subdomain
	}

	return res, diags
}

func (x *AkpInstance) UpdateFrom(p *argocdv1.Instance) diag.Diagnostics {
	diags := diag.Diagnostics{}
	d := diag.Diagnostics{}
	x.Id = types.StringValue(p.Id)
	x.Name = types.StringValue(p.Name)
	x.Version = types.StringValue(p.Version)
	x.Hostname = types.StringValue(p.Hostname)

	if p.Description == "" { // not computed
		x.Description = types.StringNull()
	} else {
		x.Description = types.StringValue(p.Description)
	}

	// ----------- Spec -------------
	var inputSpec *argocdv1.InstanceSpec
	if p.Spec == nil || p.Spec.String() == "" {
		inputSpec = &argocdv1.InstanceSpec{}
	} else {
		inputSpec = p.Spec
	}
	x.AuditExtension = types.BoolValue(inputSpec.AuditExtensionEnabled)         // computed => cannot be null
	x.BackendIpAllowList = types.BoolValue(inputSpec.BackendIpAllowListEnabled) // computed => cannot be null

	if inputSpec.ClusterCustomizationDefaults == nil || inputSpec.ClusterCustomizationDefaults.String() == "" {
		x.ClusterCustomization = types.ObjectNull(clusterCustomizationAttrTypes) // not computed => can be null
	} else {
		clusterCustomizationObject := &AkpClusterCustomization{}
		diags.Append(clusterCustomizationObject.UpdateObject(inputSpec.ClusterCustomizationDefaults)...)
		x.ClusterCustomization, d = types.ObjectValueFrom(context.Background(), clusterCustomizationAttrTypes, clusterCustomizationObject)
		diags.Append(d...)
	}

	x.DeclarativeManagement = types.BoolValue(inputSpec.DeclarativeManagementEnabled) // computed => cannot be null

	if inputSpec.Extensions == nil || len(inputSpec.Extensions) == 0 {
		x.Extensions = types.ListNull(
			types.ObjectType{
				AttrTypes: extensionInstallEntryAttrTypes,
			}, // not computed => can be null
		)
	} else {
		var extensions []*AkpArgoCDExtensionInstallEntry
		for _, entry := range inputSpec.Extensions {
			extension := &AkpArgoCDExtensionInstallEntry{}
			diags.Append(extension.UpdateObject(entry)...)
			extensions = append(extensions, extension)
		}
		x.Extensions, d = types.ListValueFrom(
			context.Background(),
			types.ObjectType{
				AttrTypes: extensionInstallEntryAttrTypes,
			},
			extensions,
		)
		diags.Append(d...)
	}

	x.ImageUpdater = types.BoolValue(inputSpec.ImageUpdaterEnabled) // computed => cannot be null

	if inputSpec.IpAllowList == nil || len(inputSpec.IpAllowList) == 0 {
		x.IpAllowList = types.ListNull(
			types.ObjectType{
				AttrTypes: iPAllowListEntryAttrTypes,
			}, // not computed => can be null
		)
	} else {
		var ipAllowList []*AkpIPAllowListEntry
		for _, entry := range inputSpec.IpAllowList {
			ipAllowListEntry := &AkpIPAllowListEntry{}
			diags.Append(ipAllowListEntry.UpdateObject(entry)...)
			ipAllowList = append(ipAllowList, ipAllowListEntry)
		}
		x.IpAllowList, d = types.ListValueFrom(
			context.Background(),
			types.ObjectType{
				AttrTypes: iPAllowListEntryAttrTypes,
			},
			ipAllowList,
		)
		diags.Append(d...)
	}

	if inputSpec.RepoServerDelegate == nil || inputSpec.RepoServerDelegate.String() == "" {
		x.RepoServerDelegate = types.ObjectNull(repoServerDelegateAttrTypes) // not computed => can be null
	} else {
		repoServerDelegateObject := &AkpRepoServerDelegate{}
		diags.Append(repoServerDelegateObject.UpdateObject(inputSpec.RepoServerDelegate)...)
		x.RepoServerDelegate, d = types.ObjectValueFrom(context.Background(), repoServerDelegateAttrTypes, repoServerDelegateObject)
		diags.Append(d...)
	}

	x.Subdomain = types.StringValue(inputSpec.Subdomain) // computed => cannot be null

	// ----------- ConfigMap -------------
	var inputConfig *argocdv1.ArgoCDConfigMap
	if p.Config == nil || p.Config.String() == "" {
		inputConfig = &argocdv1.ArgoCDConfigMap{}
	} else {
		inputConfig = p.Config
	}

	x.AdminEnabled = types.BoolValue(inputConfig.GetAdminEnabled())             // computed => cannot be null
	x.AllowAnonymousUser = types.BoolValue(inputConfig.GetAllowAnonymousUser()) // computed => cannot be null
	if inputConfig.Banner == nil || inputConfig.Banner.String() == "" {
		x.Banner = types.ObjectNull(bannerAttrTypes) // not computed => can be null
	} else {
		banner := &AkpArgoCDBanner{}
		diags.Append(banner.UpdateObject(inputConfig.Banner)...)
		x.Banner, d = types.ObjectValueFrom(context.Background(), bannerAttrTypes, &banner)
		diags.Append(d...)
	}

	if inputConfig.Chat == nil || inputConfig.Chat.String() == "" {
		x.Chat = types.ObjectNull(chatAttrTypes) // not computed => can be null
	} else {
		chat := &AkpArgoCDChat{}
		diags.Append(chat.UpdateObject(inputConfig.Chat)...)
		x.Chat, d = types.ObjectValueFrom(context.Background(), chatAttrTypes, &chat)
		diags.Append(d...)
	}

	if inputConfig.DexConfig == "" {
		x.DexConfig = types.StringNull() // not computed => can be null
	} else {
		x.DexConfig = types.StringValue(inputConfig.DexConfig)
	}

	if inputConfig.GoogleAnalytics == nil || inputConfig.GoogleAnalytics.String() == "" {
		x.GoogleAnalytics = types.ObjectNull(googleAnalyticsAttrTypes) // not computed => can be null
	} else {
		googleAnalytics := &AkpArgoCDGoogleAnalytics{}
		diags.Append(googleAnalytics.UpdateObject(inputConfig.GoogleAnalytics)...)
		x.GoogleAnalytics, d = types.ObjectValueFrom(context.Background(), googleAnalyticsAttrTypes, &googleAnalytics)
		diags.Append(d...)
	}

	x.HelmEnabled = types.BoolValue(p.Config.HelmSettings.Enabled) // computed => cannot be null
	if inputConfig.HelmSettings.GetValueFileSchemas() == "" {
		x.HelmSettings = types.ObjectNull(HelmSettingsAttrTypes) // not computed => can be null
	} else {
		helmSettings := &AkpArgoCDHelmSettings{}
		diags.Append(helmSettings.UpdateObject(inputConfig.HelmSettings)...)
		x.HelmSettings, d = types.ObjectValueFrom(context.Background(), HelmSettingsAttrTypes, &helmSettings)
		diags.Append(d...)
	}

	if inputConfig.InstanceLabelKey == "" {
		x.InstanceLabelKey = types.StringNull() // not computed => can be null
	} else {
		x.InstanceLabelKey = types.StringValue(inputConfig.InstanceLabelKey)
	}

	x.KustomizeEnabled = types.BoolValue(p.Config.KustomizeSettings.Enabled) // computed => cannot be null
	if inputConfig.KustomizeSettings.BuildOptions == "" {
		x.KustomizeSettings = types.ObjectNull(KustomizeSettingsAttrTypes) // not computed => can be null
	} else {
		kustomizeSettings := &AkpArgoCDKustomizeSettings{}
		diags.Append(kustomizeSettings.UpdateObject(inputConfig.KustomizeSettings)...)
		x.KustomizeSettings, d = types.ObjectValueFrom(context.Background(), KustomizeSettingsAttrTypes, &kustomizeSettings)
		diags.Append(d...)
	}

	if inputConfig.OidcConfig == "" {
		x.OidcConfig = types.StringNull() // not computed => can be null
	} else {
		x.OidcConfig = types.StringValue(inputConfig.OidcConfig)
	}

	if inputConfig.ResourceSettings == nil || inputConfig.ResourceSettings.String() == "" {
		x.ResourceSettings = types.ObjectNull(resourceSettingsAttrTypes) // not computed => can be null
	} else {
		resourceSettings := &AkpArgoCDResourceSettings{}
		diags.Append(resourceSettings.UpdateObject(inputConfig.ResourceSettings)...)
		x.ResourceSettings, d = types.ObjectValueFrom(context.Background(), resourceSettingsAttrTypes, &resourceSettings)
		diags.Append(d...)
	}

	if inputConfig.StatusBadge == nil || inputConfig.StatusBadge.String() == "" {
		x.StatusBadge = types.ObjectNull(statusBadgeAttrTypes) // not computed => can be null
	} else {
		statusBadge := &AkpArgoCDStatusBadge{}
		diags.Append(statusBadge.UpdateObject(inputConfig.StatusBadge)...)
		x.StatusBadge, d = types.ObjectValueFrom(context.Background(), statusBadgeAttrTypes, &statusBadge)
		diags.Append(d...)
	}

	if inputConfig.UsersSessionDuration == "" {
		x.UsersSessionDuration = types.StringNull() // not computed => can be null
	} else {
		x.UsersSessionDuration = types.StringValue(inputConfig.UsersSessionDuration)
	}

	if inputConfig.WebTerminal == nil || inputConfig.WebTerminal.String() == "" {
		x.WebTerminal = types.ObjectNull(webTerminalAttrTypes) // not computed => can be null
	} else {
		webTerminal := &AkpArgoCDWebTerminal{}
		diags.Append(webTerminal.UpdateObject(inputConfig.WebTerminal)...)
		x.WebTerminal, d = types.ObjectValueFrom(context.Background(), webTerminalAttrTypes, &webTerminal)
		diags.Append(d...)
	}

	// ----------- RBAC -------------
	var inputRbac *argocdv1.ArgoCDRBACConfigMap
	if p.RbacConfig == nil || p.RbacConfig.String() == "" {
		inputRbac = &argocdv1.ArgoCDRBACConfigMap{}
	} else {
		inputRbac = p.RbacConfig
	}

	if inputRbac.DefaultPolicy == "" {
		x.DefaultPolicy = types.StringNull() // not computed => can be null
	} else {
		x.DefaultPolicy = types.StringValue(inputRbac.DefaultPolicy)
	}

	if inputRbac.PolicyCsv == "" {
		x.PolicyCsv = types.StringNull() // not computed => can be null
	} else {
		x.PolicyCsv = types.StringValue(inputRbac.PolicyCsv)
	}

	if len(inputRbac.Scopes) == 0 {
		x.OidcScopes = types.ListNull(types.StringType) // not computed => can be null
	} else {
		var scopes []types.String
		for _, entry := range inputRbac.Scopes {
			scope := types.StringValue(entry)
			scopes = append(scopes, scope)
		}
		x.OidcScopes, diags = types.ListValueFrom(context.Background(), types.StringType, scopes)
	}

	return diags
}

func (x *AkpInstance) As(target *argocdv1.Instance) diag.Diagnostics {
	diags := diag.Diagnostics{}

	target.Name = x.Name.ValueString()
	target.Description = x.Description.ValueString()
	target.Version = x.Version.ValueString()

	// ------- Spec -------
	if target.Spec == nil {
		target.Spec = &argocdv1.InstanceSpec{}
	}
	target.Spec.AuditExtensionEnabled = x.AuditExtension.ValueBool()
	target.Spec.BackendIpAllowListEnabled = x.BackendIpAllowList.ValueBool()

	if x.ClusterCustomization.IsNull() {
		target.Spec.ClusterCustomizationDefaults = nil
	} else if !x.ClusterCustomization.IsUnknown() {
		clusterCustomizationObject := &AkpClusterCustomization{}
		if target.Spec.ClusterCustomizationDefaults != nil {
			diags.Append(clusterCustomizationObject.UpdateObject(target.Spec.ClusterCustomizationDefaults)...)
		}
		targetClusterCustomization := argocdv1.ClusterCustomization{}
		diags.Append(x.ClusterCustomization.As(context.Background(), clusterCustomizationObject, basetypes.ObjectAsOptions{})...)
		diags.Append(clusterCustomizationObject.As(&targetClusterCustomization)...)
		target.Spec.ClusterCustomizationDefaults = &targetClusterCustomization
	}

	target.Spec.DeclarativeManagementEnabled = x.DeclarativeManagement.ValueBool()

	if x.Extensions.IsNull() {
		target.Spec.Extensions = nil
	} else if !x.Extensions.IsUnknown() {
		var extensionsList []*AkpArgoCDExtensionInstallEntry
		diags.Append(x.Extensions.ElementsAs(context.Background(), &extensionsList, true)...)
		for _, extensionObject := range extensionsList {
			extension := argocdv1.ArgoCDExtensionInstallEntry{}
			diags.Append(extensionObject.As(&extension)...)
			target.Spec.Extensions = append(target.Spec.Extensions, &extension)
		}
	}

	target.Spec.ImageUpdaterEnabled = x.ImageUpdater.ValueBool()

	if x.IpAllowList.IsNull() {
		target.Spec.IpAllowList = nil
	} else if !x.IpAllowList.IsUnknown() {
		var ipAllowList []*AkpIPAllowListEntry
		diags.Append(x.IpAllowList.ElementsAs(context.Background(), &ipAllowList, true)...)
		for _, ipAllowEntryObject := range ipAllowList {
			ipAllowEntry := argocdv1.IPAllowListEntry{}
			diags.Append(ipAllowEntryObject.As(&ipAllowEntry)...)
			target.Spec.IpAllowList = append(target.Spec.IpAllowList, &ipAllowEntry)
		}
	}

	if x.RepoServerDelegate.IsNull() {
		target.Spec.RepoServerDelegate = nil
	} else if !x.RepoServerDelegate.IsUnknown() {
		repoServerDelegate := &AkpRepoServerDelegate{}
		if target.Spec.RepoServerDelegate != nil {
			diags.Append(repoServerDelegate.UpdateObject(target.Spec.RepoServerDelegate)...)
		}
		targetRepoServerDelegate := argocdv1.RepoServerDelegate{}
		diags.Append(x.RepoServerDelegate.As(context.Background(), repoServerDelegate, basetypes.ObjectAsOptions{})...)
		diags.Append(repoServerDelegate.As(&targetRepoServerDelegate)...)
		target.Spec.RepoServerDelegate = &targetRepoServerDelegate
	}

	target.Spec.Subdomain = x.Subdomain.ValueString()

	// ------- Config -------
	if target.Config == nil {
		target.Config = &argocdv1.ArgoCDConfigMap{}
	}
	target.Config.AdminEnabled = x.AdminEnabled.ValueBool()
	target.Config.AllowAnonymousUser = x.AllowAnonymousUser.ValueBool()

	if x.Banner.IsNull() {
		target.Config.Banner = nil
	} else if !x.Banner.IsUnknown() {
		banner := AkpArgoCDBanner{}
		if target.Config.Banner != nil {
			diags.Append(banner.UpdateObject(target.Config.Banner)...)
		}
		targetBanner := argocdv1.ArgoCDBannerConfig{}
		diags.Append(x.Banner.As(context.Background(), &banner, basetypes.ObjectAsOptions{})...)
		diags.Append(banner.As(&targetBanner)...)
		target.Config.Banner = &targetBanner
	}

	if x.Chat.IsNull() {
		target.Config.Chat = nil
	} else if !x.Chat.IsUnknown() {
		chat := AkpArgoCDChat{}
		if target.Config.Chat != nil {
			diags.Append(chat.UpdateObject(target.Config.Chat)...)
		}
		targetChat := argocdv1.ArgoCDAlertConfig{}
		diags.Append(x.Chat.As(context.Background(), &chat, basetypes.ObjectAsOptions{})...)
		diags.Append(chat.As(&targetChat)...)
		target.Config.Chat = &targetChat
	}

	target.Config.DexConfig = x.DexConfig.ValueString()

	if x.GoogleAnalytics.IsNull() {
		target.Config.GoogleAnalytics = nil
	} else if !x.GoogleAnalytics.IsUnknown() {
		googleAnalytics := AkpArgoCDGoogleAnalytics{}
		if target.Config.GoogleAnalytics != nil {
			diags.Append(googleAnalytics.UpdateObject(target.Config.GoogleAnalytics)...)
		}
		targetGoogleAnalytics := argocdv1.ArgoCDGoogleAnalyticsConfig{}
		diags.Append(x.GoogleAnalytics.As(context.Background(), &googleAnalytics, basetypes.ObjectAsOptions{})...)
		diags.Append(googleAnalytics.As(&targetGoogleAnalytics)...)
		target.Config.GoogleAnalytics = &targetGoogleAnalytics
	}

	targetHelmSettings := argocdv1.ArgoCDHelmSettings{
		Enabled: x.HelmEnabled.ValueBool(),
	}
	if !x.HelmSettings.IsNull() && !x.HelmSettings.IsUnknown() {
		helmSettings := AkpArgoCDHelmSettings{}
		if target.Config.HelmSettings != nil {
			diags.Append(helmSettings.UpdateObject(target.Config.HelmSettings)...)
		}
		diags.Append(x.HelmSettings.As(context.Background(), &helmSettings, basetypes.ObjectAsOptions{})...)
		diags.Append(helmSettings.As(&targetHelmSettings)...)
	}
	target.Config.HelmSettings = &targetHelmSettings

	target.Config.InstanceLabelKey = x.InstanceLabelKey.ValueString()

	targetKustomizeSettings := argocdv1.ArgoCDKustomizeSettings{
		Enabled: x.KustomizeEnabled.ValueBool(),
	}
	if !x.KustomizeSettings.IsNull() && !x.KustomizeSettings.IsUnknown() {
		kustomizeSettings := AkpArgoCDKustomizeSettings{}
		if target.Config.KustomizeSettings != nil {
			diags.Append(kustomizeSettings.UpdateObject(target.Config.KustomizeSettings)...)
		}
		diags.Append(x.KustomizeSettings.As(context.Background(), &kustomizeSettings, basetypes.ObjectAsOptions{})...)
		diags.Append(kustomizeSettings.As(&targetKustomizeSettings)...)
	}
	target.Config.KustomizeSettings = &targetKustomizeSettings

	target.Config.OidcConfig = x.OidcConfig.ValueString()

	if x.ResourceSettings.IsNull() {
		target.Config.ResourceSettings = nil
	} else if !x.ResourceSettings.IsUnknown() {
		resourceSettings := AkpArgoCDResourceSettings{}
		if target.Config.ResourceSettings != nil {
			diags.Append(resourceSettings.UpdateObject(target.Config.ResourceSettings)...)
		}
		targetResourceSettings := argocdv1.ArgoCDResourceSettings{}
		diags.Append(x.ResourceSettings.As(context.Background(), &resourceSettings, basetypes.ObjectAsOptions{})...)
		diags.Append(resourceSettings.As(&targetResourceSettings)...)
		target.Config.ResourceSettings = &targetResourceSettings
	}

	if x.StatusBadge.IsNull() {
		target.Config.StatusBadge = nil
	} else if !x.StatusBadge.IsUnknown() {
		statusBadge := AkpArgoCDStatusBadge{}
		if target.Config.StatusBadge != nil {
			diags.Append(statusBadge.UpdateObject(target.Config.StatusBadge)...)
		}
		targetStatusBadge := argocdv1.ArgoCDStatusBadgeConfig{}
		diags.Append(x.StatusBadge.As(context.Background(), &statusBadge, basetypes.ObjectAsOptions{})...)
		diags.Append(statusBadge.As(&targetStatusBadge)...)
		target.Config.StatusBadge = &targetStatusBadge
	}

	target.Config.UsersSessionDuration = x.UsersSessionDuration.ValueString()

	if x.WebTerminal.IsNull() {
		target.Config.WebTerminal = nil
	} else if !x.WebTerminal.IsUnknown() {
		webTerminal := AkpArgoCDWebTerminal{}
		if target.Config.WebTerminal != nil {
			diags.Append(webTerminal.UpdateObject(target.Config.WebTerminal)...)
		}
		targetWebTerminal := argocdv1.ArgoCDWebTerminalConfig{}
		diags.Append(x.WebTerminal.As(context.Background(), &webTerminal, basetypes.ObjectAsOptions{})...)
		diags.Append(webTerminal.As(&targetWebTerminal)...)
		target.Config.WebTerminal = &targetWebTerminal
	}

	if target.RbacConfig == nil {
		target.RbacConfig = &argocdv1.ArgoCDRBACConfigMap{}
	}
	target.RbacConfig.DefaultPolicy = x.DefaultPolicy.ValueString()
	target.RbacConfig.PolicyCsv = x.PolicyCsv.ValueString()
	if x.OidcScopes.IsNull() {
		target.RbacConfig.Scopes = nil
	} else if !x.OidcScopes.IsUnknown() {
		var scopes []string
		diags.Append(x.OidcScopes.ElementsAs(context.Background(), &scopes, true)...)
		target.RbacConfig.Scopes = scopes
	}

	return diags
}

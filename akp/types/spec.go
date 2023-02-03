package types

import (
	"context"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type AkpInstanceSpec struct {
	AuditExtension        types.Bool   `tfsdk:"audit_extension"`
	BackendIpAllowList    types.Bool   `tfsdk:"backend_ip_allow_list"`
	ClusterCustomization  types.Object `tfsdk:"cluster_customization_defaults"`
	DeclarativeManagement types.Bool   `tfsdk:"declarative_management"`
	Extensions            types.List   `tfsdk:"extensions"`
	ImageUpdater          types.Bool   `tfsdk:"image_updater"`
	IpAllowList           types.List   `tfsdk:"ip_allow_list"`
	RepoServerDelegate    types.Object `tfsdk:"repo_server_delegate"`
	Subdomain             types.String `tfsdk:"subdomain"`
}

var (
	instanceSpecAttrTypes = map[string]attr.Type{
		"audit_extension":       types.BoolType,
		"backend_ip_allow_list": types.BoolType,
		"cluster_customization_defaults": types.ObjectType{
			AttrTypes: clusterCustomizationAttrTypes,
		},
		"declarative_management": types.BoolType,
		"extensions": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: extensionInstallEntryAttrTypes,
			},
		},
		"image_updater": types.BoolType,
		"ip_allow_list": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: iPAllowListEntryAttrTypes,
			},
		},
		"repo_server_delegate": types.ObjectType{
			AttrTypes: repoServerDelegateAttrTypes,
		},
		"subdomain": types.StringType,
	}
)

func MergeSpec(state *AkpInstanceSpec, plan *AkpInstanceSpec) (*AkpInstanceSpec, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpInstanceSpec{}

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

func (x *AkpInstanceSpec) UpdateObject(p *argocdv1.InstanceSpec) diag.Diagnostics {
	diags := diag.Diagnostics{}
	var d diag.Diagnostics
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.InstanceSpec is <nil>")
		return diags
	}

	x.AuditExtension = types.BoolValue(p.AuditExtensionEnabled)
	x.BackendIpAllowList = types.BoolValue(p.BackendIpAllowListEnabled)

	if p.ClusterCustomizationDefaults == nil || p.ClusterCustomizationDefaults.String() == "" {
		x.ClusterCustomization = types.ObjectNull(clusterCustomizationAttrTypes)
	} else {
		clusterCustomizationObject := &AkpClusterCustomization{}
		diags.Append(clusterCustomizationObject.UpdateObject(p.ClusterCustomizationDefaults)...)
		x.ClusterCustomization, d = types.ObjectValueFrom(context.Background(), clusterCustomizationAttrTypes, clusterCustomizationObject)
		diags.Append(d...)
	}

	x.DeclarativeManagement = types.BoolValue(p.DeclarativeManagementEnabled)

	if p.Extensions == nil || len(p.Extensions) == 0 {
		x.Extensions = types.ListNull(
			types.ObjectType{
				AttrTypes: extensionInstallEntryAttrTypes,
			},
		)
	} else {
		var extensions []*AkpArgoCDExtensionInstallEntry
		for _, entry := range p.Extensions {
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

	x.ImageUpdater = types.BoolValue(p.ImageUpdaterEnabled)

	if p.IpAllowList == nil || len(p.IpAllowList) == 0 {
		x.IpAllowList = types.ListNull(
			types.ObjectType{
				AttrTypes: iPAllowListEntryAttrTypes,
			},
		)
	} else {
		var ipAllowList []*AkpIPAllowListEntry
		for _, entry := range p.IpAllowList {
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

	if p.RepoServerDelegate == nil || p.RepoServerDelegate.String() == "" {
		x.RepoServerDelegate = types.ObjectNull(repoServerDelegateAttrTypes)
	} else {
		repoServerDelegateObject := &AkpRepoServerDelegate{}
		diags.Append(repoServerDelegateObject.UpdateObject(p.RepoServerDelegate)...)
		x.RepoServerDelegate, d = types.ObjectValueFrom(context.Background(), repoServerDelegateAttrTypes, repoServerDelegateObject)
		diags.Append(d...)
	}

	if p.Subdomain == "" {
		x.Subdomain = types.StringNull()
	} else {
		x.Subdomain = types.StringValue(p.Subdomain)
	}

	return diags
}

func (x *AkpInstanceSpec) As(target *argocdv1.InstanceSpec) diag.Diagnostics {
	diags := diag.Diagnostics{}

	target.AuditExtensionEnabled = x.AuditExtension.ValueBool()
	target.BackendIpAllowListEnabled = x.BackendIpAllowList.ValueBool()

	if x.ClusterCustomization.IsNull() {
		target.ClusterCustomizationDefaults = nil
	} else if !x.ClusterCustomization.IsUnknown() {
		clusterCustomizationObject := &AkpClusterCustomization{}
		if target.ClusterCustomizationDefaults != nil {
			diags.Append(clusterCustomizationObject.UpdateObject(target.ClusterCustomizationDefaults)...)
		}
		targetClusterCustomization := argocdv1.ClusterCustomization{}
		diags.Append(x.ClusterCustomization.As(context.Background(), clusterCustomizationObject, basetypes.ObjectAsOptions{})...)
		diags.Append(clusterCustomizationObject.As(&targetClusterCustomization)...)
		target.ClusterCustomizationDefaults = &targetClusterCustomization
	}

	target.DeclarativeManagementEnabled = x.DeclarativeManagement.ValueBool()

	if x.Extensions.IsNull() {
		target.Extensions = nil
	} else if !x.Extensions.IsUnknown() {
		var extensionsList []*AkpArgoCDExtensionInstallEntry
		diags.Append(x.Extensions.ElementsAs(context.Background(), &extensionsList, true)...)
		for _, extensionObject := range extensionsList {
			extension := argocdv1.ArgoCDExtensionInstallEntry{}
			diags.Append(extensionObject.As(&extension)...)
			target.Extensions = append(target.Extensions, &extension)
		}
	}

	target.ImageUpdaterEnabled = x.ImageUpdater.ValueBool()

	if x.IpAllowList.IsNull() {
		target.IpAllowList = nil
	} else if !x.IpAllowList.IsUnknown() {
		var ipAllowList []*AkpIPAllowListEntry
		diags.Append(x.IpAllowList.ElementsAs(context.Background(), &ipAllowList, true)...)
		for _, ipAllowEntryObject := range ipAllowList {
			ipAllowEntry := argocdv1.IPAllowListEntry{}
			diags.Append(ipAllowEntryObject.As(&ipAllowEntry)...)
			target.IpAllowList = append(target.IpAllowList, &ipAllowEntry)
		}
	}

	if x.RepoServerDelegate.IsNull() {
		target.RepoServerDelegate = nil
	} else if !x.RepoServerDelegate.IsUnknown() {
		repoServerDelegate := &AkpRepoServerDelegate{}
		if target.RepoServerDelegate != nil {
			diags.Append(repoServerDelegate.UpdateObject(target.RepoServerDelegate)...)
		}
		targetRepoServerDelegate := argocdv1.RepoServerDelegate{}
		diags.Append(x.RepoServerDelegate.As(context.Background(), repoServerDelegate, basetypes.ObjectAsOptions{})...)
		diags.Append(repoServerDelegate.As(&targetRepoServerDelegate)...)
		target.RepoServerDelegate = &targetRepoServerDelegate
	}

	target.Subdomain = x.Subdomain.ValueString()
	return diags
}

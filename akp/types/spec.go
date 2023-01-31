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
		"audit_extension": types.BoolType,
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
		"subdomain":     types.StringType,
	}
)

func (x *AkpInstanceSpec) UpdateObject(p *argocdv1.InstanceSpec) diag.Diagnostics {
	diags := diag.Diagnostics{}
	var d diag.Diagnostics
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.InstanceSpec is <nil>")
		return diags
	}

	x.AuditExtension = types.BoolValue(p.AuditExtensionEnabled)
	x.BackendIpAllowList = types.BoolValue(p.BackendIpAllowListEnabled)

	if p.ClusterCustomizationDefaults == nil {
		x.ClusterCustomization = types.ObjectNull(clusterCustomizationAttrTypes)
	} else {
		clusterCustomizationObject := &AkpClusterCustomization{}
		diags.Append(clusterCustomizationObject.UpdateObject(p.ClusterCustomizationDefaults)...)
		x.ClusterCustomization, d = types.ObjectValueFrom(context.Background(), clusterCustomizationAttrTypes, clusterCustomizationObject)
		diags.Append(d...)
	}

	x.DeclarativeManagement = types.BoolValue(p.DeclarativeManagementEnabled)

	x.Extensions, d = types.ListValueFrom(
		context.Background(),
		types.ObjectType{
			AttrTypes: extensionInstallEntryAttrTypes,
		},
		p.Extensions,
	)
	diags.Append(d...)

	x.ImageUpdater = types.BoolValue(p.ImageUpdaterEnabled)

	x.IpAllowList, d = types.ListValueFrom(
		context.Background(),
		types.ObjectType{
			AttrTypes: iPAllowListEntryAttrTypes,
		},
		p.IpAllowList,
	)
	diags.Append(d...)

	if p.RepoServerDelegate == nil {
		x.RepoServerDelegate = types.ObjectNull(repoServerDelegateAttrTypes)
	} else {
		repoServerDelegateObject := &AkpRepoServerDelegate{}
		diags.Append(repoServerDelegateObject.UpdateObject(p.RepoServerDelegate)...)
		x.RepoServerDelegate, d = types.ObjectValueFrom(context.Background(), repoServerDelegateAttrTypes, repoServerDelegateObject)
		diags.Append(d...)
	}

	x.Subdomain = types.StringValue(p.Subdomain)
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
		targetRepoServerDelegate := argocdv1.RepoServerDelegate{}
		diags.Append(x.RepoServerDelegate.As(context.Background(), repoServerDelegate, basetypes.ObjectAsOptions{})...)
		diags.Append(repoServerDelegate.As(&targetRepoServerDelegate)...)
		target.RepoServerDelegate = &targetRepoServerDelegate
	}

	target.Subdomain = x.Subdomain.ValueString()
	return diags
}

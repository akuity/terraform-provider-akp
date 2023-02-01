package types

import (
	"context"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type AkpRepoServerDelegate struct {
	ControlPlane     types.Object `tfsdk:"control_plane"`
	ManagedCluster   types.Object `tfsdk:"managed_cluster"`
}

var (
	repoServerDelegateAttrTypes = map[string]attr.Type{
		"control_plane": types.ObjectType{
			AttrTypes: repoServerDelegateControlPlanerAttrTypes,
		},
		"managed_cluster":   types.ObjectType{
			AttrTypes: repoServerDelegateManagedClusterAttrTypes,
		},
	}
)

func (x *AkpRepoServerDelegate) UpdateObject(p *argocdv1.RepoServerDelegate) diag.Diagnostics {
	diags := diag.Diagnostics{}
	var d diag.Diagnostics
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.RepoServerDelegate is <nil>")
		return diags
	}

	if p.ControlPlane == nil {
		x.ControlPlane = types.ObjectNull(repoServerDelegateControlPlanerAttrTypes)
	} else {
		controlPlaneObject := &AkpRepoServerDelegateControlPlane{}
		diags.Append(controlPlaneObject.UpdateObject(p.ControlPlane)...)
		x.ControlPlane, d = types.ObjectValueFrom(context.Background(), repoServerDelegateControlPlanerAttrTypes, controlPlaneObject)
		diags.Append(d...)
	}

	if p.ManagedCluster == nil {
		x.ManagedCluster = types.ObjectNull(repoServerDelegateManagedClusterAttrTypes)
	} else {
		managedClusterObject := &AkpRepoServerDelegateManagedCluster{}
		diags.Append(managedClusterObject.UpdateObject(p.ManagedCluster)...)
		x.ManagedCluster, d = types.ObjectValueFrom(context.Background(), repoServerDelegateManagedClusterAttrTypes, managedClusterObject)
		diags.Append(d...)
	}

	return diags
}

func (x *AkpRepoServerDelegate) As(target *argocdv1.RepoServerDelegate) diag.Diagnostics {
	diags := diag.Diagnostics{}

	if x.ControlPlane.IsNull() {
		target.ControlPlane = nil
	} else if !x.ControlPlane.IsUnknown() {
		controlPlane := AkpRepoServerDelegateControlPlane{}
		targetControlPlane := argocdv1.RepoServerDelegateControlPlane{}
		diags.Append(x.ControlPlane.As(context.Background(), &controlPlane, basetypes.ObjectAsOptions{})...)
		diags.Append(controlPlane.As(&targetControlPlane)...)
		target.ControlPlane = &targetControlPlane
	}

	if x.ManagedCluster.IsNull() {
		target.ManagedCluster = nil
	} else if !x.ManagedCluster.IsUnknown() {
		managedCluster := AkpRepoServerDelegateManagedCluster{}
		targetManagedCluster := argocdv1.RepoServerDelegateManagedCluster{}
		diags.Append(x.ManagedCluster.As(context.Background(), managedCluster, basetypes.ObjectAsOptions{})...)
		diags.Append(managedCluster.As(&targetManagedCluster)...)
		target.ManagedCluster = &targetManagedCluster
	}


	return diags
}

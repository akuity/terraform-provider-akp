package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type AkpRepoServerDelegateControlPlane struct {}

var (
	repoServerDelegateControlPlanerAttrTypes = map[string]attr.Type{}
)

func MergeRepoServerDelegateControlPlane(state *AkpRepoServerDelegateControlPlane, plan *AkpRepoServerDelegateControlPlane) (*AkpRepoServerDelegateControlPlane, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpRepoServerDelegateControlPlane{}

	return res, diags
}

func (x *AkpRepoServerDelegateControlPlane) UpdateObject(p *argocdv1.RepoServerDelegateControlPlane) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.RepoServerDelegateControlPlane is <nil>")
	}
	return diags
}

func (x *AkpRepoServerDelegateControlPlane) As(target *argocdv1.RepoServerDelegateControlPlane) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target = nil
	return diags
}

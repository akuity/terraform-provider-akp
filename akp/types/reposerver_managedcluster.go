package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpRepoServerDelegateManagedCluster struct {
	ClusterName  types.String `tfsdk:"cluster_name"`
}

var (
	repoServerDelegateManagedClusterAttrTypes = map[string]attr.Type{
		"cluster_name": types.StringType,
	}
)

func (x *AkpRepoServerDelegateManagedCluster) UpdateObject(p *argocdv1.RepoServerDelegateManagedCluster) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.RepoServerDelegateManagedCluster is <nil>")
		return diags
	}
	if p.ClusterName == "" {
		x.ClusterName = types.StringNull()
	} else {
		x.ClusterName = types.StringValue(p.ClusterName)
	}
	return diags
}

func (x *AkpRepoServerDelegateManagedCluster) As(target *argocdv1.RepoServerDelegateManagedCluster) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.ClusterName = x.ClusterName.ValueString()
	return diags
}

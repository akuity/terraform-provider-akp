package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDExtensionInstallEntry struct {
	Id      types.String `tfsdk:"id"`
	Version types.String `tfsdk:"version"`
}

var (
	extensionInstallEntryAttrTypes = map[string]attr.Type{
		"id":      types.StringType,
		"version": types.StringType,
	}
)

func (x *AkpArgoCDExtensionInstallEntry) UpdateObject(p *argocdv1.ArgoCDExtensionInstallEntry) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ArgoCDExtensionInstallEntry is <nil>")
		return diags
	}
	if p.Id == "" {
		x.Id = types.StringNull()
	} else {
		x.Id = types.StringValue(p.Id)
	}

	if p.Version == "" {
		x.Version = types.StringNull()
	} else {
		x.Version = types.StringValue(p.Version)
	}

	return diags
}

func (x *AkpArgoCDExtensionInstallEntry) As(target *argocdv1.ArgoCDExtensionInstallEntry) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Id = x.Id.ValueString()
	target.Version = x.Version.ValueString()
	return diags
}

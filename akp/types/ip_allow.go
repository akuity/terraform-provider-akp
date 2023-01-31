package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpIPAllowListEntry struct {
	Ip          types.String `tfsdk:"ip"`
	Description types.String `tfsdk:"description"`
}

var (
	iPAllowListEntryAttrTypes = map[string]attr.Type{
		"ip":          types.StringType,
		"description": types.StringType,
	}
)

func (x *AkpIPAllowListEntry) UpdateObject(p *argocdv1.IPAllowListEntry) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.IPAllowListEntry is <nil>")
		return diags
	}
	x.Ip = types.StringValue(p.GetIp())
	x.Description = types.StringValue(p.GetDescription())
	return diags
}

func (x *AkpIPAllowListEntry) As(target *argocdv1.IPAllowListEntry) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.Ip = x.Ip.ValueString()
	target.Description = x.Description.ValueString()
	return diags
}

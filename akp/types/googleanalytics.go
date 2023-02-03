package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDGoogleAnalytics struct {
	AnonymizeUsers types.Bool   `tfsdk:"anonymize_users"`
	TrackingId     types.String `tfsdk:"tracking_id"`
}

var (
	googleAnalyticsAttrTypes = map[string]attr.Type{
		"anonymize_users": types.BoolType,
		"tracking_id":     types.StringType,
	}
)

func MergeGoogleAnalytics(state *AkpArgoCDGoogleAnalytics, plan *AkpArgoCDGoogleAnalytics) (*AkpArgoCDGoogleAnalytics, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpArgoCDGoogleAnalytics{}

	if plan.AnonymizeUsers.IsUnknown() {
		res.AnonymizeUsers = state.AnonymizeUsers
	} else if plan.AnonymizeUsers.IsNull() {
		res.AnonymizeUsers = types.BoolNull()
	} else {
		res.AnonymizeUsers = plan.AnonymizeUsers
	}

	if plan.TrackingId.IsUnknown() {
		res.TrackingId = state.TrackingId
	} else if plan.TrackingId.IsNull() {
		res.TrackingId = types.StringNull()
	} else {
		res.TrackingId = plan.TrackingId
	}

	return res, diags
}

func (x *AkpArgoCDGoogleAnalytics) UpdateObject(p *argocdv1.ArgoCDGoogleAnalyticsConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Conversion Error", "*argocdv1.ArgoCDGoogleAnalyticsConfig is <nil>")
		return diags
	}

	if p.TrackingId ==  "" {
		x.TrackingId = types.StringNull()
	} else {
		x.TrackingId = types.StringValue(p.TrackingId)
	}
	
	x.AnonymizeUsers = types.BoolValue(p.GetAnonymizeUsers())
	return diags
}

func (x *AkpArgoCDGoogleAnalytics) As(target *argocdv1.ArgoCDGoogleAnalyticsConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.TrackingId = x.TrackingId.ValueString()
	target.AnonymizeUsers = x.AnonymizeUsers.ValueBool()
	return diags
}

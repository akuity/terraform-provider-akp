package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpArgoCDGoogleAnalytics struct {
	TrackingId     types.String `tfsdk:"tracking_id"`
	AnonymizeUsers types.Bool   `tfsdk:"anonymize_users"`
}

var (
	googleAnalyticsAttrTypes = map[string]attr.Type{
		"tracking_id":     types.StringType,
		"anonymize_users": types.BoolType,
	}
)

func (x *AkpArgoCDGoogleAnalytics) UpdateObject(p *argocdv1.ArgoCDGoogleAnalyticsConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	x.TrackingId = types.StringValue(p.GetTrackingId())
	x.AnonymizeUsers = types.BoolValue(p.GetAnonymizeUsers())
	return diags
}

func (x *AkpArgoCDGoogleAnalytics) As(target *argocdv1.ArgoCDGoogleAnalyticsConfig) diag.Diagnostics {
	diags := diag.Diagnostics{}
	target.TrackingId = x.TrackingId.ValueString()
	target.AnonymizeUsers = x.AnonymizeUsers.ValueBool()
	return diags
}

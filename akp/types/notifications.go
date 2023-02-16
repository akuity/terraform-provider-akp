package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpNotifications struct {
	Secrets types.Map `tfsdk:"secrets"`
	Config  types.Map `tfsdk:"config"`
}

var (
	notificationsAttrTypes = map[string]attr.Type{
		"secrets": types.MapType{
			ElemType: types.ObjectType{AttrTypes: secretAttrTypes},
		},
		"config": types.MapType{
			ElemType: types.StringType,
		},
	}
)

func MergeNotifications(state *AkpNotifications, plan *AkpNotifications) (*AkpNotifications, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpNotifications{}

	if plan.Secrets.IsUnknown() {
		res.Secrets = state.Secrets
	} else {
		secrets, d := MergeSecrets(&state.Secrets, &plan.Secrets)
		diags.Append(d...)
		res.Secrets = *secrets
	}

	if plan.Config.IsUnknown() {
		res.Config = state.Config
	} else {
		res.Config = plan.Config
	}
	return res, diags
}

func (x *AkpNotifications) UpdateNotifications(secrets map[string]string, config map[string]string) diag.Diagnostics {
	var d diag.Diagnostics
	diags := diag.Diagnostics{}
	if len(secrets) == 0 {
		x.Secrets = types.MapNull(types.ObjectType{AttrTypes: secretAttrTypes}) // not computed => can be null
	} else {
		x.Secrets, d = MapValueFromMap(secrets)
		diags.Append(d...)
	}

	if len(config) == 0 {
		x.Config = types.MapNull(types.StringType)
	} else {
		x.Config, d = types.MapValueFrom(context.Background(),types.StringType, &config)
		diags.Append(d...)
	}

	return diags
}

func (x *AkpNotifications) PopulateSecrets(source *AkpNotifications) {
	secrets, _ := MapFromMapValue(x.Secrets)
	sourceSecrets, _ := MapFromMapValue(source.Secrets)
	for name := range secrets {
		secrets[name] = sourceSecrets[name]
	}
	x.Secrets, _ = MapValueFromMap(secrets)

}

func (x *AkpNotifications) GetSensitiveStrings() []string {
	var res []string
	secrets, _ := MapFromMapValue(x.Secrets)
	for _, value := range secrets {
		res = append(res, value)
	}
	return res
}

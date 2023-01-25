package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpInstance struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Version     types.String `tfsdk:"version"`
	Description types.String `tfsdk:"description"`
	Hostname    types.String `tfsdk:"hostname"`
}

func (x *AkpInstance) UpdateInstance(p *argocdv1.Instance) diag.Diagnostics {
	diag := diag.Diagnostics{}
	x.Id = types.StringValue(p.Id)
	x.Name = types.StringValue(p.GetName())
	x.Version = types.StringValue(p.GetVersion())
	x.Description = types.StringValue(p.GetDescription())
	x.Hostname = types.StringValue(p.GetHostname())
	return diag
}

func (x *AkpInstance) ToProto() (*argocdv1.Instance, diag.Diagnostics) {

	diagnostics := diag.Diagnostics{}
	res := &argocdv1.Instance{
		Id:          x.Id.ValueString(),
		Name:        x.Name.ValueString(),
		Description: x.Description.ValueString(),
		Version:     x.Version.ValueString(),
	}
	return res, diagnostics
}

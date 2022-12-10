package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ProtoInstance struct {
	*argocdv1.Instance
}

type AkpInstance struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Version       types.String `tfsdk:"version"`
	Description   types.String `tfsdk:"description"`
	Hostname      types.String `tfsdk:"hostname"`
}

var (
	instanceAttrTypes = map[string]attr.Type{
		"id":          types.StringType,
		"name":        types.StringType,
		"version":     types.StringType,
		"description": types.StringType,
	}
)

func (x *AkpInstance) ToProto() (*argocdv1.Instance, diag.Diagnostics) {

	diagnostics := diag.Diagnostics{}
	res := &argocdv1.Instance{
		Id:   x.Id.ValueString(),
		Name: x.Name.ValueString(),
		Data: &argocdv1.InstanceData{
			Description:   x.Description.ValueString(),
			Version:       x.Version.ValueString(),
		},
	}
	return res, diagnostics
}

func (x *ProtoInstance) FromProto() *AkpInstance {
	return &AkpInstance{
		Id:            types.StringValue(x.Id),
		Name:          types.StringValue(x.Name),
		Version:       types.StringValue(x.Data.Version),
		Description:   types.StringValue(x.Data.Description),
		Hostname:      types.StringValue(x.Hostname),
	}
}

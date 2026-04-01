package string

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type UnknownWhenCustomSizeModifier struct{}

func (m UnknownWhenCustomSizeModifier) Description(_ context.Context) string {
	return "Marks kustomization as unknown when transitioning to custom size and user did not set kustomization, since the server generates kustomization patches."
}

func (m UnknownWhenCustomSizeModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m UnknownWhenCustomSizeModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	var planSize types.String
	diags := req.Plan.GetAttribute(ctx, req.Path.ParentPath().AtName("size"), &planSize)
	if diags.HasError() || planSize.ValueString() != "custom" {
		return
	}

	if req.State.Raw.IsNull() {
		resp.PlanValue = types.StringUnknown()
		return
	}

	var stateSize types.String
	diags = req.State.GetAttribute(ctx, req.Path.ParentPath().AtName("size"), &stateSize)
	if diags.HasError() || stateSize.IsNull() || stateSize.ValueString() != "custom" {
		resp.PlanValue = types.StringUnknown()
	}
}

func UnknownWhenCustomSize() planmodifier.String {
	return UnknownWhenCustomSizeModifier{}
}

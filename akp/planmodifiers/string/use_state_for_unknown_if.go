package stringplanmodifier

import (
	"context"
	"fmt"

	akptypes "github.com/akuity/terraform-provider-akp/akp/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func UseStateForUnknownIfNotChanged() planmodifier.String {
	return useStateForUnknownIfNotChangedModifier{}
}

type useStateForUnknownIfNotChangedModifier struct {
	path path.Path
}

// Description returns a human-readable description of the plan modifier.
func (m useStateForUnknownIfNotChangedModifier) Description(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m useStateForUnknownIfNotChangedModifier) MarkdownDescription(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

func (m useStateForUnknownIfNotChangedModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Do nothing if there is a known planned value.
	if !req.PlanValue.IsUnknown() {
		return
	}

	// Do not replace on resource creation.
	if req.State.Raw.IsNull() {
		return
	}

	// Do not replace on resource destroy.
	if req.Plan.Raw.IsNull() {
		return
	}

	// Do not replace if the plan and state values are equal.
	if req.PlanValue.Equal(req.StateValue) {
		return
	}

	var state, plan akptypes.AkpInstance
	tflog.Debug(ctx, "UseStateForUnknownIfNotChangedModifier")
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	tflog.Debug(ctx, fmt.Sprintf("State: %s", state))
	d := req.Plan.Get(ctx, &plan)
	tflog.Debug(ctx, fmt.Sprintf("Plan: %s", plan))
	tflog.Debug(ctx, fmt.Sprintf("Errors: %v", d))

	if plan.Subdomain.IsUnknown() || state.Subdomain.Equal(plan.Subdomain) {
		resp.PlanValue = req.StateValue
	}

}

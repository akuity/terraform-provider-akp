package mapplanmodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// UseStateForNullUnknown is similar to the upstream UseStateForUnknown, but it also use the state if the prior state is null.
func UseStateForNullUnknown() planmodifier.Map {
	return UseStateForNullUnknownModifier{}
}

// UseStateForNullUnknownModifier implements the plan modifier.
type UseStateForNullUnknownModifier struct{}

// Description returns a human-readable description of the plan modifier.
func (m UseStateForNullUnknownModifier) Description(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m UseStateForNullUnknownModifier) MarkdownDescription(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

// PlanModifyMap implements the plan modification logic.
func (m UseStateForNullUnknownModifier) PlanModifyMap(_ context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	// Do nothing if there is a known planned value.
	if !req.PlanValue.IsUnknown() {
		return
	}

	// Do nothing if there is an unknown configuration value, otherwise interpolation gets messed up.
	if req.ConfigValue.IsUnknown() {
		return
	}

	resp.PlanValue = req.StateValue
}

package stringplanmodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

func UseStateNullForUnknown() planmodifier.String {
	return useStateNullForUnknownModifier{}
}

// useStateNullForUnknownModifier implements the plan modifier.
type useStateNullForUnknownModifier struct{}

// Description returns a human-readable description of the plan modifier.
func (m useStateNullForUnknownModifier) Description(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m useStateNullForUnknownModifier) MarkdownDescription(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

// PlanModifyString implements the plan modification logic.
func (m useStateNullForUnknownModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {

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

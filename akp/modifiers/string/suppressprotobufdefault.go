package string

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// SuppressProtobufDefault returns a plan modifier that suppresses diffs when
// the config is null and the state holds a protobuf default value (empty string).
// This prevents false diffs after import when the API returns "" for an unset
// Optional field, but the user's config doesn't specify the field.
//
// Removal semantics are preserved: if the state holds a non-default value
// (meaning the user previously set it), the modifier does not fire, allowing
// the diff to proceed normally.
func SuppressProtobufDefault() planmodifier.String {
	return suppressProtobufDefaultModifier{}
}

type suppressProtobufDefaultModifier struct{}

func (m suppressProtobufDefaultModifier) Description(_ context.Context) string {
	return "Suppresses diffs when config is null and state holds a protobuf default value (empty string)."
}

func (m suppressProtobufDefaultModifier) MarkdownDescription(_ context.Context) string {
	return "Suppresses diffs when config is null and state holds a protobuf default value (empty string)."
}

func (m suppressProtobufDefaultModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Only intervene when config is null (user didn't set the field).
	if !req.ConfigValue.IsNull() {
		return
	}

	// Only suppress when state is the protobuf default (empty string).
	// Non-default state values (e.g. user previously set "1.2.3") are NOT
	// suppressed, so removal diffs work correctly.
	if req.StateValue.IsNull() || req.StateValue.ValueString() != "" {
		return
	}

	resp.PlanValue = req.StateValue
}

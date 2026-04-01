package bool

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// SuppressProtobufDefault returns a plan modifier that suppresses diffs when
// the config is null and the state holds a protobuf default value (false).
// This prevents false diffs after import when the API returns false for an unset
// Optional field, but the user's config doesn't specify the field.
//
// Removal semantics are preserved: if the state holds a non-default value
// (meaning the user previously set it), the modifier does not fire, allowing
// the diff to proceed normally.
func SuppressProtobufDefault() planmodifier.Bool {
	return suppressProtobufDefaultModifier{}
}

type suppressProtobufDefaultModifier struct{}

func (m suppressProtobufDefaultModifier) Description(_ context.Context) string {
	return "Suppresses diffs when config is null and state holds a protobuf default value (false)."
}

func (m suppressProtobufDefaultModifier) MarkdownDescription(_ context.Context) string {
	return "Suppresses diffs when config is null and state holds a protobuf default value (false)."
}

func (m suppressProtobufDefaultModifier) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	if req.StateValue.IsNull() || req.StateValue.ValueBool() {
		return
	}

	resp.PlanValue = req.StateValue
}

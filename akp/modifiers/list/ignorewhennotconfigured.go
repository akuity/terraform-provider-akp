package listplanmodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// IgnoreWhenNotConfigured returns a plan modifier that preserves the state value
// when the configuration value is null (not specified). This allows other resources
// to manage the field without causing drift detection.
//
// This is useful for deprecating fields that are being migrated to separate resources.
// When the field is not specified in the configuration, the plan modifier will use
// the state value, allowing external resources to manage it. When the field is
// specified in the configuration, it works normally for backward compatibility.
func IgnoreWhenNotConfigured() planmodifier.List {
	return ignoreWhenNotConfiguredModifier{}
}

// ignoreWhenNotConfiguredModifier implements the plan modifier.
type ignoreWhenNotConfiguredModifier struct{}

// Description returns a human-readable description of the plan modifier.
func (m ignoreWhenNotConfiguredModifier) Description(_ context.Context) string {
	return "When the configuration value is null, the state value is preserved to allow external management."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m ignoreWhenNotConfiguredModifier) MarkdownDescription(_ context.Context) string {
	return "When the configuration value is null, the state value is preserved to allow external management."
}

// PlanModifyList implements the plan modification logic.
func (m ignoreWhenNotConfiguredModifier) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	// If the config value is null (not specified in configuration),
	// preserve the state value to allow external management
	if req.ConfigValue.IsNull() {
		resp.PlanValue = req.StateValue
		return
	}

	// If the config value is set, use it normally (backward compatibility)
	// The default behavior will apply, which uses the planned value
}

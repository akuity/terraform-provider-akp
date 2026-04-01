package mapplanmodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SuppressNonConfigKeys returns a plan modifier that carries forward state
// values for map keys that exist in state but not in the user's config.
// This prevents false diffs after import or when the API returns a superset
// of keys compared to what the user manages.
func SuppressNonConfigKeys() planmodifier.Map {
	return suppressNonConfigKeysModifier{}
}

type suppressNonConfigKeysModifier struct{}

func (m suppressNonConfigKeysModifier) Description(_ context.Context) string {
	return "Suppresses plan diffs for map keys that exist in state but are not present in the user's configuration."
}

func (m suppressNonConfigKeysModifier) MarkdownDescription(_ context.Context) string {
	return "Suppresses plan diffs for map keys that exist in state but are not present in the user's configuration."
}

func (m suppressNonConfigKeysModifier) PlanModifyMap(_ context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if req.ConfigValue.IsUnknown() {
		return
	}

	// When config is null (user didn't set the field), preserve the state.
	// Server-managed maps like argocd_cm always have default keys; removing
	// them from state would cause a false diff every plan cycle.
	if req.ConfigValue.IsNull() {
		if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
			resp.PlanValue = req.StateValue
		}
		return
	}

	// If state is null or unknown, this is a fresh create — nothing to preserve.
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	configElems := req.ConfigValue.Elements()
	stateElems := req.StateValue.Elements()

	if len(stateElems) <= len(configElems) {
		return
	}

	planMatchesConfig := true
	for k, cv := range configElems {
		sv, inState := stateElems[k]
		if !inState {
			planMatchesConfig = false
			break
		}
		if cv.(types.String).ValueString() != sv.(types.String).ValueString() {
			planMatchesConfig = false
			break
		}
	}

	if planMatchesConfig {
		resp.PlanValue = req.StateValue
	}
}

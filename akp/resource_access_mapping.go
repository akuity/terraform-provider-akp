package akp

import (
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
)

// Canonical response-mapping helpers for the hand-written "access" resources
// (team, api_key, workspace_member). These resources mix Computed attributes
// (ids, timestamps, member counts) with Optional-not-Computed identifiers and
// lists (custom_roles, user_email, team_name). The helpers below apply only to
// the latter: their post-apply value must equal the planned value exactly, so
// the mapper must not change the operator's null-vs-empty encoding or overwrite
// config with a server-canonicalized value. These mirror the rules the
// reflection state-builder (setSliceFromAPI/sameElements) applies for the large
// CRD-backed resources.

// applyStringList overwrites a list attribute only when the server returns
// elements; an empty response keeps `current` so an explicit `[]` stays `[]`
// and an omitted field stays null (a nil slice reflects as a null list).
// Trade-off: out-of-band removal of *all* elements isn't surfaced as drift
// (partial changes still are).
func applyStringList(current []tftypes.String, server []string) []tftypes.String {
	if len(server) == 0 {
		return current
	}
	out := make([]tftypes.String, 0, len(server))
	for _, v := range server {
		out = append(out, tftypes.StringValue(v))
	}
	return out
}

// hydrateIfUnset keeps the operator's value, taking the server value only when
// current is unset (the import case). Needed for scalars the backend
// canonicalizes (e.g. lowercased emails), where echoing the server value back
// over config would diverge from it.
func hydrateIfUnset(current tftypes.String, server string) tftypes.String {
	if current.IsNull() || current.IsUnknown() {
		return tftypes.StringValue(server)
	}
	return current
}

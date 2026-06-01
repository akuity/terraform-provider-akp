package akp

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// customRolePolicyValidator parses the Casbin policy at plan time and reports
// the structural mistakes the server would otherwise surface as opaque
// Internal/InvalidArgument errors after apply: wrong field count per
// directive, unknown object names, unknown verbs, predefined-role subjects.
//
// Scope-correctness (org-only objects in a workspace policy and vice versa)
// is left to the server — it requires cross-attribute access (workspace) the
// per-attribute validator interface doesn't have.
type customRolePolicyValidator struct{}

func (customRolePolicyValidator) Description(context.Context) string {
	return "Casbin policy must use 4 fields per `p` line (sub, obj, act, resource) and 2 per `g` line; objects must be known and subjects must not collide with predefined roles."
}

func (v customRolePolicyValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (customRolePolicyValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, msg := range validateCustomRolePolicy(req.ConfigValue.ValueString()) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid custom_role policy", msg)
	}
}

// validateCustomRolePolicy returns one error message per problem found. It
// is implemented as a package-private function so unit tests can exercise it
// without constructing the framework's request/response plumbing.
func validateCustomRolePolicy(policy string) []string {
	var errs []string
	for lineNum, raw := range strings.Split(policy, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := splitAndTrim(line, ",")
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "p":
			if len(fields) != 5 {
				errs = append(errs, fmt.Sprintf(
					"line %d: `p` policy expects 4 fields (sub, obj, act, resource), got %d in %q",
					lineNum+1, len(fields)-1, line,
				))
				continue
			}
			sub, obj, act := fields[1], fields[2], fields[3]
			if _, ok := predefinedRoleSubjects[sub]; ok {
				errs = append(errs, fmt.Sprintf(
					"line %d: subject %q is a predefined role; pick a different name",
					lineNum+1, sub,
				))
			}
			if _, ok := knownPolicyObjects[obj]; !ok {
				errs = append(errs, fmt.Sprintf(
					"line %d: unknown object %q; expected one of the org/workspace permission objects (see internal/services/permissions)",
					lineNum+1, obj,
				))
			}
			if _, ok := knownPolicyVerbs[act]; !ok {
				errs = append(errs, fmt.Sprintf(
					"line %d: unknown verb %q; expected get, create, update, delete, or *",
					lineNum+1, act,
				))
			}
		case "g":
			if len(fields) != 3 {
				errs = append(errs, fmt.Sprintf(
					"line %d: `g` grouping expects 2 fields (subject, role), got %d in %q",
					lineNum+1, len(fields)-1, line,
				))
			}
		default:
			errs = append(errs, fmt.Sprintf(
				"line %d: unrecognized directive %q; only `p` and `g` are supported",
				lineNum+1, fields[0],
			))
		}
	}
	return errs
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

// knownPolicyObjects is the union of every Object exported from
// internal/services/permissions plus `*` (wildcard). Synced manually to keep
// the provider decoupled from the server-internal package — validation here
// is a pre-flight nicety; the server is the source of truth. If a new
// permission object is added in internal/services/permissions, add it here
// too. Drift only costs a slightly worse error message (the server will
// still reject the unknown object on apply).
var knownPolicyObjects = map[string]struct{}{
	"*": {},
	// Legacy/back-compat objects
	"organization/instances":       {},
	"organization/kargo-instances": {},
	"instance/clusters":            {},
	// Org-scoped objects
	"organization":                      {},
	"organization/apikeys":              {},
	"organization/custom-role":          {},
	"organization/audit-log":            {},
	"organization/workspaces":           {},
	"organization/admins":               {},
	"organization/billing":              {},
	"organization/members":              {},
	"organization/member-role":          {},
	"organization/sso-configuration":    {},
	"organization/oidc-map":             {},
	"organization/teams":                {},
	"organization/notification-configs": {},
	"organization/kubernetes-dashboard": {},
	"organization/ai-support-engineer":  {},
	"team/members":                      {},
	// Workspace-scoped objects
	"workspace/instances":             {},
	"workspace/instance/clusters":     {},
	"workspace/member-role":           {},
	"workspace/members":               {},
	"workspace/apikeys":               {},
	"workspace/custom-role":           {},
	"workspace/kargo-instances":       {},
	"workspace/kargo-instance/agents": {},
}

var knownPolicyVerbs = map[string]struct{}{
	"get":    {},
	"create": {},
	"update": {},
	"delete": {},
	"*":      {},
}

var predefinedRoleSubjects = map[string]struct{}{
	"role:organization/member": {},
	"role:organization/admin":  {},
	"role:organization/owner":  {},
	"role:workspace/admin":     {},
	"role:workspace/member":    {},
}

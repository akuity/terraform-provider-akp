package akp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCustomRolePolicy(t *testing.T) {
	cases := map[string]struct {
		policy   string
		errCount int
		errMatch string // substring; only checked when non-empty
	}{
		"valid p rule (scope not enforced client-side)": {
			policy: `p, role:cluster-reader, workspace/instance/clusters, get, ws/inst/cl`,
		},
		"valid org wildcard": {
			policy: `p, role:foo, organization/apikeys, *, *`,
		},
		"valid with comments and blank lines": {
			policy: `
# manage org api keys
p, role:foo, organization/apikeys, get, *

p, role:foo, organization/apikeys, create, *
`,
		},
		"valid grouping": {
			policy: `g, alice, role:foo`,
		},
		"five-field p line (the '..., allow' bug)": {
			policy:   `p, role:foo, organization/apikeys, get, *, allow`,
			errCount: 1,
			errMatch: "expects 4 fields",
		},
		"three-field p line": {
			policy:   `p, role:foo, organization/apikeys, get`,
			errCount: 1,
			errMatch: "expects 4 fields",
		},
		"unknown object": {
			policy:   `p, role:foo, applications, get, *`,
			errCount: 1,
			errMatch: `unknown object "applications"`,
		},
		"unknown verb": {
			policy:   `p, role:foo, organization/apikeys, read, *`,
			errCount: 1,
			errMatch: `unknown verb "read"`,
		},
		"predefined role subject (org owner)": {
			policy:   `p, role:organization/owner, organization/apikeys, get, *`,
			errCount: 1,
			errMatch: "predefined role",
		},
		"predefined role subject (workspace admin)": {
			policy:   `p, role:workspace/admin, workspace/instances, get, *`,
			errCount: 1,
			errMatch: "predefined role",
		},
		"one-field g line": {
			policy:   `g, alice`,
			errCount: 1,
			errMatch: "expects 2 fields",
		},
		"unknown directive": {
			policy:   `q, foo, bar`,
			errCount: 1,
			errMatch: `unrecognized directive "q"`,
		},
		"reports all errors": {
			policy: `p, role:foo, applications, read, *
p, role:bar, organization/apikeys, get, *, allow`,
			errCount: 3, // unknown object + unknown verb on line 1, field-count on line 2
		},
		"empty policy is valid": {
			policy: "",
		},
		"line numbering accounts for blank prefix": {
			policy:   "\n\np, role:foo, applications, get, *",
			errCount: 1,
			errMatch: "line 3:",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			errs := validateCustomRolePolicy(tc.policy)
			require.Len(t, errs, tc.errCount, "errs: %v", errs)
			if tc.errMatch != "" {
				var combined strings.Builder
				for _, e := range errs {
					combined.WriteString(e + "\n")
				}
				require.Contains(t, combined.String(), tc.errMatch)
			}
		})
	}
}

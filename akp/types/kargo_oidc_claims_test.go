package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

var kargoClaimAttrTypes = map[string]attr.Type{
	"values": tftypes.SetType{ElemType: tftypes.StringType},
}

func kargoClaim(values attr.Value) attr.Value {
	return tftypes.ObjectValueMust(kargoClaimAttrTypes, map[string]attr.Value{"values": values})
}

func kargoClaimValues(vals ...string) attr.Value {
	elems := make([]attr.Value, 0, len(vals))
	for _, v := range vals {
		elems = append(elems, tftypes.StringValue(v))
	}
	return tftypes.SetValueMust(tftypes.StringType, elems)
}

func kargoClaimsMap(claims map[string]attr.Value) tftypes.Map {
	return tftypes.MapValueMust(tftypes.ObjectType{AttrTypes: kargoClaimAttrTypes}, claims)
}

// setPredefinedAccount attaches acct to the named OIDC predefined account.
func setPredefinedAccount(oidc *KargoOidcConfig, account string, acct *KargoPredefinedAccountData) {
	switch account {
	case "admin":
		oidc.AdminAccount = acct
	case "viewer":
		oidc.ViewerAccount = acct
	case "user":
		oidc.UserAccount = acct
	case "projectCreator":
		oidc.ProjectCreatorAccount = acct
	}
}

func getPredefinedAccount(oidc *KargoOidcConfig, account string) *KargoPredefinedAccountData {
	switch account {
	case "admin":
		return oidc.AdminAccount
	case "viewer":
		return oidc.ViewerAccount
	case "user":
		return oidc.UserAccount
	case "projectCreator":
		return oidc.ProjectCreatorAccount
	}
	return nil
}

// The platform collapses an applied `values = []` to nil before persisting it (dedupArray),
// so the export reads it back as `"values": null`. Refresh has to keep the planned empty set
// for those claims, or every apply fails with "Provider produced inconsistent result after
// apply" — while a claim the platform really did clear still has to surface as drift.
func TestBuildStateFromAPI_KargoOIDCClaimValues(t *testing.T) {
	cases := map[string]struct {
		planClaims tftypes.Map
		apiClaims  map[string]any
		wantClaims tftypes.Map
	}{
		"applied empty set survives the null the API reports": {
			planClaims: kargoClaimsMap(map[string]attr.Value{
				"email":  kargoClaim(kargoClaimValues()),
				"groups": kargoClaim(kargoClaimValues("app_github_employee")),
				"sub":    kargoClaim(kargoClaimValues()),
			}),
			apiClaims: map[string]any{
				"email":  map[string]any{"values": nil},
				"groups": map[string]any{"values": []any{"app_github_employee"}},
				"sub":    map[string]any{"values": nil},
			},
			wantClaims: kargoClaimsMap(map[string]attr.Value{
				"email":  kargoClaim(kargoClaimValues()),
				"groups": kargoClaim(kargoClaimValues("app_github_employee")),
				"sub":    kargoClaim(kargoClaimValues()),
			}),
		},
		"a null in config stays null": {
			planClaims: kargoClaimsMap(map[string]attr.Value{
				"email": kargoClaim(tftypes.SetNull(tftypes.StringType)),
			}),
			apiClaims: map[string]any{
				"email": map[string]any{"values": nil},
			},
			wantClaims: kargoClaimsMap(map[string]attr.Value{
				"email": kargoClaim(tftypes.SetNull(tftypes.StringType)),
			}),
		},
		"a populated claim cleared out of band is still drift": {
			planClaims: kargoClaimsMap(map[string]attr.Value{
				"groups": kargoClaim(kargoClaimValues("app_github_employee")),
			}),
			apiClaims: map[string]any{
				"groups": map[string]any{"values": nil},
			},
			wantClaims: kargoClaimsMap(map[string]attr.Value{
				"groups": kargoClaim(tftypes.SetNull(tftypes.StringType)),
			}),
		},
		"a claim changed out of band follows the API": {
			planClaims: kargoClaimsMap(map[string]attr.Value{
				"groups": kargoClaim(kargoClaimValues()),
			}),
			apiClaims: map[string]any{
				"groups": map[string]any{"values": []any{"added-elsewhere"}},
			},
			wantClaims: kargoClaimsMap(map[string]attr.Value{
				"groups": kargoClaim(kargoClaimValues("added-elsewhere")),
			}),
		},
		"a claim only the API knows about is taken as-is": {
			planClaims: kargoClaimsMap(map[string]attr.Value{
				"email": kargoClaim(kargoClaimValues()),
			}),
			apiClaims: map[string]any{
				"email":  map[string]any{"values": nil},
				"groups": map[string]any{"values": nil},
			},
			wantClaims: kargoClaimsMap(map[string]attr.Value{
				"email":  kargoClaim(kargoClaimValues()),
				"groups": kargoClaim(tftypes.SetNull(tftypes.StringType)),
			}),
		},
	}

	for name, tc := range cases {
		for _, account := range []string{"admin", "viewer", "user", "projectCreator"} {
			t.Run(account+"/"+name, func(t *testing.T) {
				oidc := &KargoOidcConfig{Enabled: tftypes.BoolValue(true)}
				setPredefinedAccount(oidc, account, &KargoPredefinedAccountData{Claims: tc.planClaims})
				plan := &Kargo{Spec: KargoSpec{OidcConfig: oidc}}
				state := DeepCopy(plan)

				apiMap := map[string]any{"spec": map[string]any{"oidcConfig": map[string]any{
					"enabled":           true,
					account + "Account": map[string]any{"claims": tc.apiClaims},
				}}}

				var diags diag.Diagnostics
				diags.Append(BuildStateFromAPI(context.Background(), apiMap, state, plan, KargoReverseOverridesMap, KargoReverseRenamesMap, "kargo")...)
				require.False(t, diags.HasError(), diags)

				got := getPredefinedAccount(state.Spec.OidcConfig, account)
				require.NotNil(t, got)
				require.Equal(t, tc.wantClaims, got.Claims)
			})
		}
	}
}

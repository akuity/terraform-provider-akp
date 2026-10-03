//go:build !acc

package akp

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

var kargoClaimAttrTypes = map[string]attr.Type{
	"values": tftypes.SetType{ElemType: tftypes.StringType},
}

// emptyValuesClaims is the claim map an `oidc_config` account carries after a config that
// applies `values = []` to every claim.
func emptyValuesClaims(t *testing.T, groups []string) tftypes.Map {
	t.Helper()
	groupElems := make([]attr.Value, 0, len(groups))
	for _, g := range groups {
		groupElems = append(groupElems, tftypes.StringValue(g))
	}
	claim := func(values attr.Value) attr.Value {
		return tftypes.ObjectValueMust(kargoClaimAttrTypes, map[string]attr.Value{"values": values})
	}
	return tftypes.MapValueMust(tftypes.ObjectType{AttrTypes: kargoClaimAttrTypes}, map[string]attr.Value{
		"email":  claim(tftypes.SetValueMust(tftypes.StringType, []attr.Value{})),
		"groups": claim(tftypes.SetValueMust(tftypes.StringType, groupElems)),
		"sub":    claim(tftypes.SetValueMust(tftypes.StringType, []attr.Value{})),
	})
}

// nullValuesClaims is what the export returns for those claims: the platform stores an
// applied empty list as nil (dedupArray), so only a populated claim comes back with values.
func nullValuesClaims(groups []string) map[string]any {
	groupValues := make([]any, 0, len(groups))
	for _, g := range groups {
		groupValues = append(groupValues, g)
	}
	var groupAttr any
	if len(groupValues) > 0 {
		groupAttr = groupValues
	}
	return map[string]any{
		"email":  map[string]any{"values": nil},
		"groups": map[string]any{"values": groupAttr},
		"sub":    map[string]any{"values": nil},
	}
}

// An `oidc_config` account claim applied as `values = []` must still read back as an empty
// set, for all four predefined accounts. The platform reports it as null, and letting that
// null into state makes the applied value disagree with the plan, which fails the apply with
// "Provider produced inconsistent result after apply" (issue #12697).
func TestKargoOIDCEmptyClaimValuesRoundTrip(t *testing.T) {
	ctx := context.Background()
	groups := []string{"app_github_employee"}

	instance := &types.KargoInstance{
		Name: tftypes.StringValue("example"),
		Kargo: &types.Kargo{
			Spec: types.KargoSpec{
				Version: tftypes.StringValue("v1.7.0"),
				OidcConfig: &types.KargoOidcConfig{
					Enabled:               tftypes.BoolValue(true),
					AdminAccount:          &types.KargoPredefinedAccountData{Claims: emptyValuesClaims(t, nil)},
					ViewerAccount:         &types.KargoPredefinedAccountData{Claims: emptyValuesClaims(t, groups)},
					UserAccount:           &types.KargoPredefinedAccountData{Claims: emptyValuesClaims(t, nil)},
					ProjectCreatorAccount: &types.KargoPredefinedAccountData{Claims: emptyValuesClaims(t, nil)},
				},
			},
		},
	}

	kargo, err := structpb.NewStruct(map[string]any{
		"spec": map[string]any{
			"version":           "v1.7.0",
			"kargoInstanceSpec": map[string]any{},
			"oidcConfig": map[string]any{
				"enabled":               true,
				"adminAccount":          map[string]any{"claims": nullValuesClaims(nil)},
				"viewerAccount":         map[string]any{"claims": nullValuesClaims(groups)},
				"userAccount":           map[string]any{"claims": nullValuesClaims(nil)},
				"projectCreatorAccount": map[string]any{"claims": nullValuesClaims(nil)},
			},
		},
	})
	require.NoError(t, err)

	var diags diag.Diagnostics
	require.NoError(t, instance.Update(ctx, &diags, &kargov1.ExportKargoInstanceResponse{Kargo: kargo}, false))
	require.Empty(t, diags)

	oidc := instance.Kargo.Spec.OidcConfig
	require.NotNil(t, oidc)
	for name, account := range map[string]*types.KargoPredefinedAccountData{
		"admin_account":           oidc.AdminAccount,
		"viewer_account":          oidc.ViewerAccount,
		"user_account":            oidc.UserAccount,
		"project_creator_account": oidc.ProjectCreatorAccount,
	} {
		require.NotNil(t, account, name)
		wantGroups := []string(nil)
		if name == "viewer_account" {
			wantGroups = groups
		}
		require.Equal(t, emptyValuesClaims(t, wantGroups), account.Claims, name)
	}
}

// The data source has no prior value to fall back on, so the null the platform reports for an
// empty claim stays null there.
func TestKargoOIDCEmptyClaimValuesDataSource(t *testing.T) {
	ctx := context.Background()
	instance := &types.KargoInstance{Name: tftypes.StringValue("example")}
	kargo, err := structpb.NewStruct(map[string]any{
		"spec": map[string]any{
			"version":           "v1.7.0",
			"kargoInstanceSpec": map[string]any{},
			"oidcConfig": map[string]any{
				"enabled":      true,
				"adminAccount": map[string]any{"claims": nullValuesClaims(nil)},
			},
		},
	})
	require.NoError(t, err)

	var diags diag.Diagnostics
	require.NoError(t, instance.Update(ctx, &diags, &kargov1.ExportKargoInstanceResponse{Kargo: kargo}, true))
	require.Empty(t, diags)

	claims := instance.Kargo.Spec.OidcConfig.AdminAccount.Claims.Elements()
	require.Contains(t, claims, "email")
	values, ok := claims["email"].(tftypes.Object).Attributes()["values"].(tftypes.Set)
	require.True(t, ok)
	require.True(t, values.IsNull())
}

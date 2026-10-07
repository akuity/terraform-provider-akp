//go:build !acc

package akp

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	accesscontrolv1 "github.com/akuity/api-client-go/pkg/api/gen/accesscontrol/v1"
	serviceaccountv1 "github.com/akuity/api-client-go/pkg/api/gen/serviceaccount/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added or removed on the service
// account types. Update the schema attributes accordingly.
func TestNoNewServiceAccountFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.ServiceAccount]().NumField(), len(getServiceAccountAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ServiceAccountOIDCBinding]().NumField(), len(getServiceAccountOIDCBindingAttributes()))
}

func TestServiceAccountSpec(t *testing.T) {
	match := func(entries map[string]string) tftypes.Map {
		elements := map[string]attr.Value{}
		for k, v := range entries {
			elements[k] = tftypes.StringValue(v)
		}
		return tftypes.MapValueMust(tftypes.StringType, elements)
	}
	plan := &types.ServiceAccount{
		Description: tftypes.StringValue("ci"),
		Permissions: &types.ApiKeyPermissions{Roles: []tftypes.String{tftypes.StringValue("member")}},
		OIDCBinding: &types.ServiceAccountOIDCBinding{
			IssuerID: tftypes.StringValue("iss-1"),
			Match:    match(map[string]string{"sub": "repo:acme/app:.*", "ref": "refs/heads/main"}),
		},
		Disabled: tftypes.BoolValue(false),
	}
	spec, err := serviceAccountSpec(context.Background(), plan)
	require.NoError(t, err)
	assert.Equal(t, "iss-1", spec.GetOidcBinding().GetIssuerId())
	assert.Equal(t, map[string]string{"sub": "repo:acme/app:.*", "ref": "refs/heads/main"}, spec.GetOidcBinding().GetMatch())
	assert.Equal(t, []string{"member"}, spec.GetPermissions().GetRoles())

	plan.OIDCBinding.Match = match(map[string]string{"repository": "acme/app"})
	_, err = serviceAccountSpec(context.Background(), plan)
	require.ErrorContains(t, err, "sub claim")

	plan.Permissions = &types.ApiKeyPermissions{}
	_, err = serviceAccountSpec(context.Background(), plan)
	require.ErrorContains(t, err, "roles")
}

func TestApplyServiceAccountResponseDescription(t *testing.T) {
	sa := &serviceaccountv1.ServiceAccount{Id: "sa-1", Permissions: &accesscontrolv1.Permissions{Roles: []string{"member"}}}
	t.Run("omitted stays null", func(t *testing.T) {
		data := &types.ServiceAccount{Description: tftypes.StringNull()}
		require.NoError(t, applyServiceAccountResponse(context.Background(), data, sa))
		assert.True(t, data.Description.IsNull())
	})
	t.Run("set is refreshed from the server", func(t *testing.T) {
		data := &types.ServiceAccount{Description: tftypes.StringValue("old")}
		require.NoError(t, applyServiceAccountResponse(context.Background(), data, sa))
		assert.Equal(t, "", data.Description.ValueString())
		sa.Description = "ci"
		data.Description = tftypes.StringNull()
		require.NoError(t, applyServiceAccountResponse(context.Background(), data, sa))
		assert.Equal(t, "ci", data.Description.ValueString())
	})
}

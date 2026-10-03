//go:build !acc

package akp

import (
	"reflect"
	"testing"

	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added or removed on the OIDC issuer
// type. Update the schema attributes accordingly.
func TestNoNewOIDCIssuerFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.OIDCIssuer]().NumField(), len(getOIDCIssuerAttributes()))
}

func TestOIDCIssuerSpecKeySource(t *testing.T) {
	base := func() *types.OIDCIssuer {
		return &types.OIDCIssuer{
			Name:            tftypes.StringValue("gh"),
			IssuerURL:       tftypes.StringValue("https://token.actions.githubusercontent.com"),
			TokenTTLSeconds: tftypes.Int64Value(3600),
		}
	}
	t.Run("discovery needs nothing else", func(t *testing.T) {
		plan := base()
		plan.KeySource = tftypes.StringValue(oidcKeySourceDiscovery)
		spec, err := oidcIssuerSpec(plan)
		require.NoError(t, err)
		assert.NotNil(t, spec.GetKeySource().GetDiscovery())
	})
	t.Run("attributes of another key source are rejected", func(t *testing.T) {
		plan := base()
		plan.KeySource = tftypes.StringValue(oidcKeySourceDiscovery)
		plan.JWKSURI = tftypes.StringValue("https://example.com/jwks")
		_, err := oidcIssuerSpec(plan)
		require.ErrorContains(t, err, "jwks_uri must be unset")
		plan.JWKSURI = tftypes.StringNull()
		plan.StaticJWKS = tftypes.StringValue(`{"keys":[]}`)
		_, err = oidcIssuerSpec(plan)
		require.ErrorContains(t, err, "static_jwks must be unset")
	})
	t.Run("jwks_uri requires the url", func(t *testing.T) {
		plan := base()
		plan.KeySource = tftypes.StringValue(oidcKeySourceJWKSURI)
		_, err := oidcIssuerSpec(plan)
		require.ErrorContains(t, err, "jwks_uri is required")
		plan.JWKSURI = tftypes.StringValue("https://example.com/jwks")
		spec, err := oidcIssuerSpec(plan)
		require.NoError(t, err)
		assert.Equal(t, "https://example.com/jwks", spec.GetKeySource().GetJwksUri())
	})
	t.Run("static_jwks requires a json object", func(t *testing.T) {
		plan := base()
		plan.KeySource = tftypes.StringValue(oidcKeySourceStaticJWKS)
		_, err := oidcIssuerSpec(plan)
		require.ErrorContains(t, err, "static_jwks is required")
		plan.StaticJWKS = tftypes.StringValue("[]")
		_, err = oidcIssuerSpec(plan)
		require.ErrorContains(t, err, "JSON object")
		plan.StaticJWKS = tftypes.StringValue(`{"keys":[{"kty":"RSA","kid":"k1"}]}`)
		spec, err := oidcIssuerSpec(plan)
		require.NoError(t, err)
		assert.Len(t, spec.GetKeySource().GetStaticJwks().GetFields()["keys"].GetListValue().GetValues(), 1)
	})
}

func TestApplyJSONString(t *testing.T) {
	mine := tftypes.StringValue("{\n  \"keys\": [{\"kid\": \"k1\", \"kty\": \"RSA\"}]\n}")
	same := `{"keys":[{"kty":"RSA","kid":"k1"}]}`
	assert.Equal(t, mine, applyJSONString(mine, same), "equivalent JSON keeps the operator's text")
	assert.Equal(t, tftypes.StringValue(`{"keys":[]}`), applyJSONString(mine, `{"keys":[]}`), "a real change takes the server's")
	assert.Equal(t, tftypes.StringValue(same), applyJSONString(tftypes.StringNull(), same), "import takes the server's")
}

package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/suite"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
)

const (
	federationPlannedJWKS = `{"keys":[{"kty":"RSA","kid":"a","n":"xyz","e":"AQAB"}]}`
	// The same key set as Postgres JSONB returns it: keys reordered, spaces added.
	federationStoredJWKS = `{"keys": [{"e": "AQAB", "n": "xyz", "kid": "a", "kty": "RSA"}]}`
	federationOtherJWKS  = `{"keys": [{"e": "AQAB", "n": "other", "kid": "b", "kty": "RSA"}]}`
	federationIssuer     = "https://issuer.example.com"
)

type FederationStateSuite struct {
	suite.Suite
}

func TestFederationStateSuite(t *testing.T) {
	suite.Run(t, new(FederationStateSuite))
}

type federationStateCase struct {
	name        string
	importing   bool
	apiJWKS     string
	wantJWKS    types.String
	wantJWKSURL types.String
}

func federationStateCases() []federationStateCase {
	return []federationStateCase{
		{
			name:        "equivalent key set keeps the planned spelling",
			apiJWKS:     federationStoredJWKS,
			wantJWKS:    types.StringValue(federationPlannedJWKS),
			wantJWKSURL: types.StringNull(),
		},
		{
			name:        "different key set shows the API value",
			apiJWKS:     federationOtherJWKS,
			wantJWKS:    types.StringValue(federationOtherJWKS),
			wantJWKSURL: types.StringNull(),
		},
		{
			name:        "import keeps an unset jwks_url null",
			importing:   true,
			apiJWKS:     federationStoredJWKS,
			wantJWKS:    types.StringValue(federationStoredJWKS),
			wantJWKSURL: types.StringNull(),
		},
	}
}

func (s *FederationStateSuite) TestCluster() {
	for _, tc := range federationStateCases() {
		s.Run(tc.name, func() {
			ctx := context.Background()
			plan := &Cluster{Spec: &ClusterSpec{Data: ClusterData{
				Federation: &ClusterFederation{
					Issuer:  types.StringValue(federationIssuer),
					Jwks:    types.StringValue(federationPlannedJWKS),
					JwksUrl: types.StringNull(),
				},
			}}}
			got := &Cluster{}
			// Import reads into the state it refreshes, with no spec yet.
			if tc.importing {
				ctx = WithReadContext(ctx)
				plan = got
			}
			var diags diag.Diagnostics
			got.Update(ctx, &diags, &argocdv1.Cluster{
				Name: "c",
				Data: &argocdv1.ClusterData{
					Federation: &argocdv1.ClusterFederation{Enabled: true, Issuer: federationIssuer, Jwks: tc.apiJWKS},
				},
			}, plan)

			s.Require().False(diags.HasError(), diags)
			s.Require().NotNil(got.Spec.Data.Federation)
			s.Equal(types.StringValue(federationIssuer), got.Spec.Data.Federation.Issuer)
			s.Equal(tc.wantJWKS, got.Spec.Data.Federation.Jwks)
			s.Equal(tc.wantJWKSURL, got.Spec.Data.Federation.JwksUrl)
		})
	}
}

func (s *FederationStateSuite) TestKargoAgent() {
	for _, tc := range federationStateCases() {
		s.Run(tc.name, func() {
			ctx := context.Background()
			plan := &KargoAgent{Spec: &KargoAgentSpec{Data: KargoAgentData{
				Federation: &KargoAgentFederation{
					Issuer:                   types.StringValue(federationIssuer),
					Jwks:                     types.StringValue(federationPlannedJWKS),
					JwksUrl:                  types.StringNull(),
					PromotionServiceAccounts: []types.String{types.StringValue("promoter")},
				},
			}}}
			got := &KargoAgent{}
			// Import reads into the state it refreshes, with no spec yet.
			if tc.importing {
				ctx = WithReadContext(ctx)
				plan = got
			}
			var diags diag.Diagnostics
			got.Update(ctx, &diags, &kargov1.KargoAgent{
				Name: "a",
				Data: &kargov1.KargoAgentData{
					Federation: &kargov1.KargoAgentFederation{
						Enabled:                  true,
						Issuer:                   federationIssuer,
						Jwks:                     tc.apiJWKS,
						PromotionServiceAccounts: []string{"promoter"},
					},
				},
			}, plan)

			s.Require().False(diags.HasError(), diags)
			s.Require().NotNil(got.Spec.Data.Federation)
			s.Equal(types.StringValue(federationIssuer), got.Spec.Data.Federation.Issuer)
			s.Equal(tc.wantJWKS, got.Spec.Data.Federation.Jwks)
			s.Equal(tc.wantJWKSURL, got.Spec.Data.Federation.JwksUrl)
			s.Equal([]types.String{types.StringValue("promoter")}, got.Spec.Data.Federation.PromotionServiceAccounts)
		})
	}
}

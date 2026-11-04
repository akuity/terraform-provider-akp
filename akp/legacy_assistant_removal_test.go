//go:build !acc

package akp

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/suite"
)

type LegacyAssistantRemovalSuite struct {
	suite.Suite
}

func TestLegacyAssistantRemovalSuite(t *testing.T) {
	suite.Run(t, new(LegacyAssistantRemovalSuite))
}

func (s *LegacyAssistantRemovalSuite) TestInstanceSchemaOmitsLegacyAssistant() {
	instance := instanceSchema().Attributes
	spec := instance["argocd"].(schema.SingleNestedAttribute).Attributes["spec"].(schema.SingleNestedAttribute).Attributes["instance_spec"].(schema.SingleNestedAttribute)
	s.NotContains(spec.Attributes, "assistant_extension_enabled")
}

func (s *LegacyAssistantRemovalSuite) TestOldStateDropsLegacyAssistant() {
	for name, legacyValue := range map[string]any{"enabled": true, "disabled": false, "null": nil} {
		s.Run(name, func() {
			ctx := s.T().Context()
			server := providerserver.NewProtocol6(New("test")())()
			legacyState := map[string]any{
				"id":   "existing-instance-id",
				"name": "existing-instance",
				"argocd": map[string]any{
					"spec": map[string]any{
						"version": "v3.4.4",
						"instance_spec": map[string]any{
							"assistant_extension_enabled": legacyValue,
							"audit_extension_enabled":     true,
						},
					},
				},
			}
			stateJSON, err := json.Marshal(legacyState)
			s.Require().NoError(err)
			// The framework's same-version upgrade drops attributes absent from
			// the current schema. Exercise the real provider RPC so removing the
			// field does not require users to edit or re-import their state.
			resp, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
				TypeName: "akp_instance",
				Version:  0,
				RawState: &tfprotov6.RawState{JSON: stateJSON},
			})
			s.Require().NoError(err)
			s.Require().Empty(resp.Diagnostics)
			s.Require().NotNil(resp.UpgradedState)
			value, err := resp.UpgradedState.Unmarshal(instanceSchema().Type().TerraformType(ctx))
			s.Require().NoError(err)
			var upgraded, argo, argoSpec, spec map[string]tftypes.Value
			s.Require().NoError(value.As(&upgraded))
			s.Equal(tftypes.NewValue(tftypes.String, legacyState["id"]), upgraded["id"])
			s.Equal(tftypes.NewValue(tftypes.String, legacyState["name"]), upgraded["name"])
			s.Require().NoError(upgraded["argocd"].As(&argo))
			s.Require().NoError(argo["spec"].As(&argoSpec))
			s.Equal(tftypes.NewValue(tftypes.String, "v3.4.4"), argoSpec["version"])
			s.Require().NoError(argoSpec["instance_spec"].As(&spec))
			s.NotContains(spec, "assistant_extension_enabled")
			s.Equal(tftypes.NewValue(tftypes.Bool, true), spec["audit_extension_enabled"])
		})
	}
}

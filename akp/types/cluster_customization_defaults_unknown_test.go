package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/suite"
)

// Optional+Computed fields left unset in config are unknown in the create plan.
// If the API response omits them, the state must not carry that unknown into
// the post-apply value, or Terraform rejects the apply with "unknown value".
type ClusterCustomizationDefaultsUnknownSuite struct {
	suite.Suite
}

func TestClusterCustomizationDefaultsUnknownSuite(t *testing.T) {
	suite.Run(t, new(ClusterCustomizationDefaultsUnknownSuite))
}

func (s *ClusterCustomizationDefaultsUnknownSuite) TestPlanUnknownNeverLeaksIntoState() {
	attrTypes := map[string]attr.Type{
		"auto_upgrade_disabled": types.BoolType,
		"app_replication":       types.BoolType,
	}

	cases := []struct {
		name   string
		apiCCD map[string]any
		planAR attr.Value
		wantAR attr.Value
	}{
		{
			name:   "API omits unset computed field, plan unknown -> null",
			apiCCD: map[string]any{"autoUpgradeDisabled": false},
			planAR: types.BoolUnknown(),
			wantAR: types.BoolNull(),
		},
		{
			name:   "API returns computed field, plan unknown -> API value",
			apiCCD: map[string]any{"autoUpgradeDisabled": false, "appReplication": true},
			planAR: types.BoolUnknown(),
			wantAR: types.BoolValue(true),
		},
		{
			name:   "API omits field, plan known -> plan value kept",
			apiCCD: map[string]any{"autoUpgradeDisabled": false},
			planAR: types.BoolValue(false),
			wantAR: types.BoolValue(false),
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			planCCD := types.ObjectValueMust(attrTypes, map[string]attr.Value{
				"auto_upgrade_disabled": types.BoolValue(false),
				"app_replication":       tc.planAR,
			})
			plan := &ArgoCD{}
			plan.Spec.InstanceSpec.ClusterCustomizationDefaults = planCCD

			apiMap := map[string]any{
				"spec": map[string]any{
					"instanceSpec": map[string]any{
						"clusterCustomizationDefaults": tc.apiCCD,
					},
				},
			}

			a := &ArgoCD{}
			var diags diag.Diagnostics
			diags.Append(BuildStateFromAPI(context.Background(), apiMap, a, plan, ReverseOverridesMap, ReverseRenamesMap, "spec")...)
			s.Require().False(diags.HasError(), "%v", diags)

			ccd := a.Spec.InstanceSpec.ClusterCustomizationDefaults
			s.Require().False(ccd.IsUnknown(), "cluster_customization_defaults object is unknown")
			s.Require().False(ccd.IsNull(), "cluster_customization_defaults object is null")
			s.Equal(tc.wantAR, ccd.Attributes()["app_replication"])
		})
	}
}

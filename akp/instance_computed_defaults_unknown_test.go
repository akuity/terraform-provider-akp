package akp

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwtypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/structpb"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

const (
	clusterCustomizationDefaultsPath = "argocd.spec.instance_spec.cluster_customization_defaults"
	agentCustomizationDefaultsPath   = "kargo.spec.kargo_instance_spec.agent_customization_defaults"
)

// Terraform plans every Optional+Computed attribute the config leaves unset as unknown on
// create. Whatever the export omits must still come back known, or the apply fails with
// "unknown value after apply" and taints the instance.
type InstanceComputedDefaultsUnknownSuite struct {
	suite.Suite
}

func TestInstanceComputedDefaultsUnknownSuite(t *testing.T) {
	suite.Run(t, new(InstanceComputedDefaultsUnknownSuite))
}

func (s *InstanceComputedDefaultsUnknownSuite) TestCreateStateHasNoUnknowns() {
	ctx := context.Background()
	autoscalerPath := clusterCustomizationDefaultsPath + ".autoscaler_config"
	appControllerPath := autoscalerPath + ".application_controller"
	resources := func(path string) fwtypes.Object {
		return s.plannedObject(ctx, path, map[string]attr.Value{
			"cpu":    fwtypes.StringValue("1"),
			"memory": fwtypes.StringValue("1Gi"),
		})
	}

	cases := []struct {
		name       string
		set        map[string]attr.Value
		exported   map[string]any
		wantValues map[string]attr.Value
	}{
		{
			name:     "only auto_upgrade_disabled set",
			set:      map[string]attr.Value{"auto_upgrade_disabled": fwtypes.BoolValue(false)},
			exported: map[string]any{"autoUpgradeDisabled": false},
			wantValues: map[string]attr.Value{
				"auto_upgrade_disabled":    fwtypes.BoolValue(false),
				"app_replication":          fwtypes.BoolNull(),
				"redis_tunneling":          fwtypes.BoolNull(),
				"server_side_diff_enabled": fwtypes.BoolNull(),
				"connectivity":             fwtypes.StringNull(),
				"custom_ca_bundle":         fwtypes.StringNull(),
			},
		},
		{
			// autoscaler_config itself is not computed, so it is planned known while its
			// unset repo_server child is unknown.
			name: "unknown nested inside a known object the export omits",
			set: map[string]attr.Value{
				"size": fwtypes.StringValue("auto"),
				"autoscaler_config": s.plannedObject(ctx, autoscalerPath, map[string]attr.Value{
					"application_controller": s.plannedObject(ctx, appControllerPath, map[string]attr.Value{
						"resource_minimum": resources(appControllerPath + ".resource_minimum"),
						"resource_maximum": resources(appControllerPath + ".resource_maximum"),
					}),
				}),
			},
			exported: map[string]any{"size": "CLUSTER_SIZE_AUTO"},
			wantValues: map[string]attr.Value{
				"size":            fwtypes.StringValue("auto"),
				"app_replication": fwtypes.BoolNull(),
			},
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			instance := &types.Instance{
				Name: fwtypes.StringValue("example"),
				ArgoCD: &types.ArgoCD{Spec: types.ArgoCDSpec{
					Version: fwtypes.StringValue("v3.4.4-ak.92"),
					InstanceSpec: types.InstanceSpec{
						ClusterCustomizationDefaults: s.plannedObject(ctx, clusterCustomizationDefaultsPath, tc.set),
					},
				}},
			}
			exported, err := structpb.NewStruct(map[string]any{
				"spec": map[string]any{
					"version":      "v3.4.4-ak.92",
					"instanceSpec": map[string]any{"clusterCustomizationDefaults": tc.exported},
				},
			})
			s.Require().NoError(err)

			var diags diag.Diagnostics
			s.Require().NoError(instance.Update(ctx, &diags, &argocdv1.ExportInstanceResponse{Argocd: exported}, false, nil))
			s.Require().False(diags.HasError(), "%v", diags)

			ccd := instance.ArgoCD.Spec.InstanceSpec.ClusterCustomizationDefaults
			s.Empty(unknownPaths("cluster_customization_defaults", ccd))
			for name, want := range tc.wantValues {
				s.Equal(want, ccd.Attributes()[name], name)
			}
		})
	}
}

// Kargo models agent_customization_defaults as Go pointer structs, so an unknown reaches state
// differently: the plan pointer is copied whole when the export omits the block.
func (s *InstanceComputedDefaultsUnknownSuite) TestKargoCreateStateHasNoUnknowns() {
	ctx := context.Background()
	for _, name := range []string{"connectivity", "custom_ca_bundle", "size"} {
		s.Require().True(types.RegisteredComputedFields[agentCustomizationDefaultsPath+"."+name], "%s must be computed for this plan shape", name)
	}

	cases := []struct {
		name     string
		exported map[string]any
	}{
		{name: "export returns only auto_upgrade_disabled", exported: map[string]any{
			"agentCustomizationDefaults": map[string]any{"autoUpgradeDisabled": false},
		}},
		{name: "export omits the block", exported: map[string]any{}},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			instance := &types.KargoInstance{
				Name: fwtypes.StringValue("example"),
				Kargo: &types.Kargo{Spec: types.KargoSpec{
					Version: fwtypes.StringValue("v1.7.0"),
					KargoInstanceSpec: types.KargoInstanceSpec{
						AgentCustomizationDefaults: &types.KargoAgentCustomization{
							AutoUpgradeDisabled: fwtypes.BoolValue(false),
							Kustomization:       fwtypes.StringNull(),
							Connectivity:        fwtypes.StringUnknown(),
							CustomCaBundle:      fwtypes.StringUnknown(),
							Size:                fwtypes.StringUnknown(),
						},
					},
				}},
			}
			exported, err := structpb.NewStruct(map[string]any{
				"spec": map[string]any{"version": "v1.7.0", "kargoInstanceSpec": tc.exported},
			})
			s.Require().NoError(err)

			var diags diag.Diagnostics
			s.Require().NoError(instance.Update(ctx, &diags, &kargov1.ExportKargoInstanceResponse{Kargo: exported}, false, nil))
			s.Require().False(diags.HasError(), "%v", diags)

			s.Empty(structUnknownPaths("agent_customization_defaults", reflect.ValueOf(instance.Kargo.Spec.KargoInstanceSpec.AgentCustomizationDefaults)))
		})
	}
}

// structUnknownPaths lists the tfsdk paths of unknown values in a Go state struct.
func structUnknownPaths(path string, v reflect.Value) []string {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return nil
		}
		return structUnknownPaths(path, v.Elem())
	case reflect.Struct:
		if av, ok := reflect.TypeAssert[attr.Value](v); ok {
			return unknownPaths(path, av)
		}
		var out []string
		for i := range v.NumField() {
			if tag := v.Type().Field(i).Tag.Get("tfsdk"); tag != "" && tag != "-" {
				out = append(out, structUnknownPaths(path+"."+tag, v.Field(i))...)
			}
		}
		return out
	}
	return nil
}

// plannedObject builds the object Terraform plans on create for a config that sets only `set`
// at path: other computed attributes are unknown and the rest are null.
func (s *InstanceComputedDefaultsUnknownSuite) plannedObject(ctx context.Context, path string, set map[string]attr.Value) fwtypes.Object {
	attrTypes := types.RegisteredObjectAttrTypes[path]
	s.Require().NotEmpty(attrTypes, "no registered attribute types for %s", path)
	attrs := make(map[string]attr.Value, len(attrTypes))
	for name, t := range attrTypes {
		if v, ok := set[name]; ok {
			attrs[name] = v
			continue
		}
		raw := tftypes.NewValue(t.TerraformType(ctx), nil)
		if types.RegisteredComputedFields[path+"."+name] {
			raw = tftypes.NewValue(t.TerraformType(ctx), tftypes.UnknownValue)
		}
		v, err := t.ValueFromTerraform(ctx, raw)
		s.Require().NoError(err)
		attrs[name] = v
	}
	return fwtypes.ObjectValueMust(attrTypes, attrs)
}

func unknownPaths(path string, v attr.Value) []string {
	if v.IsUnknown() {
		return []string{path}
	}
	var out []string
	if obj, ok := v.(fwtypes.Object); ok && !obj.IsNull() {
		for name, child := range obj.Attributes() {
			out = append(out, unknownPaths(path+"."+name, child)...)
		}
	}
	return out
}

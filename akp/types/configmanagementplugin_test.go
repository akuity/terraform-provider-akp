package types

import (
	"context"
	"maps"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func buildCMPStruct(t *testing.T, name string, apiMap map[string]any) *structpb.Struct {
	t.Helper()
	fullMap := map[string]any{
		"metadata": map[string]any{
			"name": name,
			"annotations": map[string]any{
				"akuity.io/enabled": "true",
				"akuity.io/image":   "quay.io/my-plugin:latest",
			},
		},
	}
	maps.Copy(fullMap, apiMap)
	s, err := structpb.NewStruct(fullMap)
	require.NoError(t, err)
	return s
}

func TestToConfigManagementPluginsTFModel_ImportStoresAPIValues(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	cmpStruct := buildCMPStruct(t, "my-plugin", map[string]any{
		"spec": map[string]any{
			"version": "",
			"generate": map[string]any{
				"command": []any{"generate.sh"},
			},
			"preserveFileMode": false,
		},
	})

	// Simulate import: oldCMPs is empty (no prior state).
	// The Read path stores API values directly. Protobuf defaults like
	// "" and false are stored as actual values (not null) because plan
	// is nil during import. The SuppressProtobufDefault plan modifier
	// on the version field handles the diff suppression at plan time.
	result := ToConfigManagementPluginsTFModel(ctx, &diags, []*structpb.Struct{cmpStruct}, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
	require.Contains(t, result, "my-plugin")

	cmp := result["my-plugin"]
	assert.True(t, cmp.Enabled.ValueBool())
	assert.Equal(t, "quay.io/my-plugin:latest", cmp.Image.ValueString())
	require.NotNil(t, cmp.Spec)
	// version="" is stored as StringValue("") — the plan modifier suppresses the diff
	assert.Equal(t, "", cmp.Spec.Version.ValueString())
}

func TestToConfigManagementPluginsTFModel_ImportPreservesNonDefaultValues(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	cmpStruct := buildCMPStruct(t, "my-plugin", map[string]any{
		"spec": map[string]any{
			"version": "1.2.3",
			"generate": map[string]any{
				"command": []any{"generate.sh"},
			},
			"preserveFileMode": true,
		},
	})

	result := ToConfigManagementPluginsTFModel(ctx, &diags, []*structpb.Struct{cmpStruct}, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
	require.Contains(t, result, "my-plugin")

	cmp := result["my-plugin"]
	require.NotNil(t, cmp.Spec)
	assert.Equal(t, "1.2.3", cmp.Spec.Version.ValueString())
	assert.True(t, cmp.Spec.PreserveFileMode.ValueBool())
}

func TestToConfigManagementPluginsTFModel_ExistingStatePreservesPlanValues(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	cmpStruct := buildCMPStruct(t, "my-plugin", map[string]any{
		"spec": map[string]any{
			"version": "",
			"generate": map[string]any{
				"command": []any{"generate.sh"},
			},
		},
	})

	oldCMPs := map[string]*ConfigManagementPlugin{
		"my-plugin": {
			Enabled: types.BoolValue(true),
			Image:   types.StringValue("quay.io/my-plugin:latest"),
			Spec: &PluginSpec{
				Version: types.StringNull(),
				Generate: &Command{
					Command: []types.String{types.StringValue("generate.sh")},
				},
			},
		},
	}

	result := ToConfigManagementPluginsTFModel(ctx, &diags, []*structpb.Struct{cmpStruct}, oldCMPs)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
	require.Contains(t, result, "my-plugin")

	cmp := result["my-plugin"]
	require.NotNil(t, cmp.Spec)
	// With existing state, version stays null (plan preserves null for protobuf defaults)
	assert.True(t, cmp.Spec.Version.IsNull(), "expected version to be null, got %q", cmp.Spec.Version.ValueString())
}

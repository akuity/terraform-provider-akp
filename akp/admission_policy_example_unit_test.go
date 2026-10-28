package akp

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	plugincel "k8s.io/apiserver/pkg/admission/plugin/cel"
	"k8s.io/apiserver/pkg/admission/plugin/policy/validating"
	"k8s.io/apiserver/pkg/cel/environment"
)

func TestAdmissionPolicyExample(t *testing.T) {
	testCases := map[string]struct {
		spec    string
		allowed bool
	}{
		"auto-sync disabled": {`{"syncPolicy":{"automated":{"enabled":false,"selfHeal":true}}}`, false},
		"auto-sync enabled":  {`{"syncPolicy":{"automated":{"enabled":true,"selfHeal":true}}}`, true},
		"enabled omitted":    {`{"syncPolicy":{"automated":{"selfHeal":true}}}`, true},
		"enabled null":       {`{"syncPolicy":{"automated":{"enabled":null,"selfHeal":true}}}`, true},
		"self-heal disabled": {`{"syncPolicy":{"automated":{"enabled":true,"selfHeal":false}}}`, false},
		"self-heal omitted":  {`{"syncPolicy":{"automated":{"enabled":true}}}`, false},
		"empty automated":    {`{"syncPolicy":{"automated":{}}}`, false},
		"automated omitted":  {`{"syncPolicy":{}}`, false},
		"syncPolicy omitted": {`{}`, false},
	}
	compiler := plugincel.NewCompiler(environment.MustBaseEnvSet(environment.DefaultCompatibilityVersion()))
	for _, path := range []string{
		"../examples/resources/akp_instance/argocd-manifests/admission-policy.yaml",
		"../docs/guides/v0.18-upgrading.md",
	} {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			manifest := string(data)
			if strings.HasSuffix(path, ".md") {
				var found bool
				_, manifest, found = strings.Cut(manifest, "```yaml\n")
				require.True(t, found)
				manifest, _, found = strings.Cut(manifest, "\n```")
				require.True(t, found)
			}
			var policy admissionregistrationv1.ValidatingAdmissionPolicy
			require.NoError(t, yaml.NewYAMLOrJSONDecoder(strings.NewReader(manifest), 4096).Decode(&policy))
			require.Len(t, policy.Spec.Validations, 1)
			result := compiler.CompileCELExpression(
				&validating.ValidationCondition{Expression: policy.Spec.Validations[0].Expression},
				plugincel.OptionalVariableDeclarations{}, environment.NewExpressions)
			require.Nil(t, result.Error)
			for name, tc := range testCases {
				t.Run(name, func(t *testing.T) {
					var spec map[string]any
					require.NoError(t, json.Unmarshal([]byte(tc.spec), &spec))
					value, _, err := result.Program.Eval(map[string]any{"object": map[string]any{"spec": spec}})
					require.NoError(t, err)
					assert.Equal(t, tc.allowed, value.Value())
				})
			}
		})
	}
}

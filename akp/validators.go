package akp

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// We can't generate the validators for a schema because they require possible complex types.
// Therefore, we need to manually attach validators to the resource schemas here. Each resource
// schema will have its own function to attach validators.
func init() {
	attachResourceValidators()
	attachKargoAgentResourceValidators()
}

var (
	minClusterNameLength  = 3
	maxClusterNameLength  = 50
	minInstanceNameLength = 3
	maxInstanceNameLength = 50
	minNamespaceLength    = 3
	maxNamespaceLength    = 63

	resourceNameRegex            = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
	resourceNameRegexDescription = "resource name must consist of lower case alphanumeric characters, digits or '-', and must start with an alphanumeric character, and end with an alphanumeric character or a digit"
)

// setStringValidators sets string validators on a schema attribute at the specified path. The path
// is a slice of strings representing the nested attribute path. For example,
// `[]string{"kube_config", "exec", "api_version"}` would set validators on the api_version attribute
// nested within kube_config.exec. This function will panic if the path is invalid or if the
// attribute is not a string attribute.
func setStringValidators(s *schema.Schema, path []string, validators []validator.String) {
	if len(path) == 0 {
		return
	}

	if len(path) == 1 {
		// Base case: set validators on the target attribute
		attr := s.Attributes[path[0]].(schema.StringAttribute)
		attr.Validators = validators
		s.Attributes[path[0]] = attr
		return
	}

	// Recursive case: navigate deeper into nested attributes
	nestedAttr := s.Attributes[path[0]].(schema.SingleNestedAttribute)
	nestedSchema := &schema.Schema{Attributes: nestedAttr.Attributes}
	setStringValidators(nestedSchema, path[1:], validators)
	nestedAttr.Attributes = nestedSchema.Attributes
	s.Attributes[path[0]] = nestedAttr
}

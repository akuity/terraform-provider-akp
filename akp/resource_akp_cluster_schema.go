package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

func (r *AkpClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = types.ClusterResourceSchema
}

func attachResourceValidators() {
	// Use the recursive helper function to set validators without manual tree walking
	setStringValidators(&types.ClusterResourceSchema, []string{"name"}, []validator.String{
		stringvalidator.LengthBetween(minClusterNameLength, maxClusterNameLength),
		stringvalidator.RegexMatches(resourceNameRegex, resourceNameRegexDescription),
	})

	setStringValidators(&types.ClusterResourceSchema, []string{"namespace"}, []validator.String{
		stringvalidator.LengthBetween(minNamespaceLength, maxNamespaceLength),
		stringvalidator.RegexMatches(resourceNameRegex, resourceNameRegexDescription),
	})

	setStringValidators(&types.ClusterResourceSchema, []string{"kube_config", "exec", "api_version"}, []validator.String{
		execAPIVersionValidator{},
	})
}

// execAPIVersionValidator emits a warning if user specifies v1alpha1.
type execAPIVersionValidator struct{}

func (v execAPIVersionValidator) Description(ctx context.Context) string {
	return "Warn if api_version == client.authentication.k8s.io/v1alpha1"
}

func (v execAPIVersionValidator) MarkdownDescription(ctx context.Context) string {
	return "Warns that v1alpha1 of the client authentication API is deprecated and will be removed in v1.24+."
}

func (v execAPIVersionValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.Equal(tftypes.StringValue("client.authentication.k8s.io/v1alpha1")) {
		resp.Diagnostics.AddWarning(
			"Deprecated API Version",
			"v1alpha1 of the client authentication API is deprecated; use v1beta1 or above. "+
				"It will be removed in Kubernetes client versions 1.24 and above.",
		)
	}
}

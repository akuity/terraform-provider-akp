package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

func (r *AkpKargoAgentResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = types.KargoAgentResourceSchema
}

func attachKargoAgentResourceValidators() {
	setStringValidators(&types.KargoAgentResourceSchema, []string{"name"}, []validator.String{
		stringvalidator.LengthBetween(minClusterNameLength, maxClusterNameLength),
		stringvalidator.RegexMatches(resourceNameRegex, resourceNameRegexDescription),
	})
}

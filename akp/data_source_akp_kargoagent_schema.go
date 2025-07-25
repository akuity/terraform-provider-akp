package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

func (a *AkpKargoAgentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = types.KargoAgentDataSourceSchema
}

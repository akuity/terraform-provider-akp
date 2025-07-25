package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

func (a *AkpKargoAgentsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = types.KargoAgentsDataSourceSchema
}

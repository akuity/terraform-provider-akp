package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

func (d *AkpClustersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = types.ClustersDataSourceSchema
}

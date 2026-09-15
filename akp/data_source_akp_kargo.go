package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &AkpKargoDataSource{}

func NewAkpKargoDataSource() datasource.DataSource {
	return &AkpKargoDataSource{}
}

// AkpKargoDataSource defines the data source implementation.
type AkpKargoDataSource struct {
	BaseDataSource
}

func (k *AkpKargoDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kargo_instance"
}

func (k *AkpKargoDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := toDataSourceAttributes(getAKPKargoInstanceAttributes(), "name")
	// Preserve the data source's public shape: unreturned secrets must not add
	// sensitivity to existing outputs of the whole object.
	delete(attrs, "kargo_secret")
	oidcConfig := attrs["kargo"].(schema.SingleNestedAttribute).Attributes["spec"].(schema.SingleNestedAttribute).Attributes["oidc_config"].(schema.SingleNestedAttribute)
	delete(oidcConfig.Attributes, "dex_config_secret")
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets information about a Kargo instance",
		Attributes:          attrs,
	}
}

func (k *AkpKargoDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading an Instance Datasource")
	var data types.KargoInstanceDataSource
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = k.AuthCtx(ctx)

	instance := &types.KargoInstance{Name: data.Name}
	if err := refreshKargoState(ctx, &resp.Diagnostics, k.akpCli, instance, k.akpCli.OrgId, true); err != nil {
		resp.Diagnostics.AddError("Failed to refresh kargo state", err.Error())
		return
	}
	data = types.NewKargoInstanceDataSourceModel(instance)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

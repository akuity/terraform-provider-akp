package akp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	argocdv1alpha1 "github.com/akuity/terraform-provider-akp/akp/apis/argocd/v1alpha1"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &AkpConfigManagementPluginsDataSource{}

func NewAkpConfigManagementPluginsDataSource() datasource.DataSource {
	return &AkpConfigManagementPluginsDataSource{}
}

// AkpConfigManagementPluginsDataSource defines the data source implementation.
type AkpConfigManagementPluginsDataSource struct {
	akpCli *AkpCli
}

func (d *AkpConfigManagementPluginsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_management_plugins"
}

func (d *AkpConfigManagementPluginsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	akpCli, ok := req.ProviderData.(*AkpCli)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *AkpCli, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	d.akpCli = akpCli
}

func (d *AkpConfigManagementPluginsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading a ConfigManagementPlugins Datasource")
	var data types.ConfigManagementPlugins

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, d.akpCli.Cred.Scheme(), d.akpCli.Cred.Credential())

	exportReq := &argocdv1.ExportInstanceRequest{
		OrganizationId: d.akpCli.OrgId,
		IdType:         idv1.Type_ID,
		Id:             data.InstanceID.ValueString(),
	}
	tflog.Debug(ctx, fmt.Sprintf("Export ConfigManagementPlugin request: %s", exportReq))
	exportResp, err := d.akpCli.Cli.ExportInstance(ctx, exportReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to export instance response, got error: %s", err))
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Export ConfigManagementPlugin response: %s", exportResp))

	//data.ID = data.InstanceID
	cmps := exportResp.GetConfigManagementPlugins()
	for _, cmp := range cmps {
		var apiCMP *argocdv1alpha1.ConfigManagementPlugin
		err = marshal.RemarshalTo(cmp.AsMap(), &apiCMP)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get ConfigManagementPlugin. %s", err))
			return
		}
		stateCMP := types.ConfigManagementPlugin{
			InstanceID: data.InstanceID,
		}
		stateCMP.Update(ctx, &resp.Diagnostics, apiCMP)
		data.Plugins = append(data.Plugins, stateCMP)
	}
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

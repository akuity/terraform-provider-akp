package akp

import (
	"context"
	"fmt"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	ctxutil "github.com/akuity/api-client-go/pkg/utils/context"
	akptypes "github.com/akuity/terraform-provider-akp/akp/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &AkpInstanceDataSource{}

func NewAkpInstanceDataSource() datasource.DataSource {
	return &AkpInstanceDataSource{}
}

// AkpInstanceDataSource defines the data source implementation.
type AkpInstanceDataSource struct {
	akpCli *AkpCli
}

func (d *AkpInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (d *AkpInstanceDataSource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Find an Argo CD instance by its name",

		Attributes: map[string]tfsdk.Attribute{
			"id": {
				MarkdownDescription: "Instance ID",
				Type:                types.StringType,
				Computed:            true,
			},
			"name": {
				MarkdownDescription: "Instance Name",
				Type:                types.StringType,
				Required:            true,
			},
			"description": {
				MarkdownDescription: "Instance Description",
				Type:                types.StringType,
				Computed:            true,
			},
			"hostname": {
				MarkdownDescription: "Instance Hostname",
				Type:                types.StringType,
				Computed:            true,
			},
			"version": {
				MarkdownDescription: "Argo CD Version",
				Type:                types.StringType,
				Computed:            true,
			},
		},
	}, nil
}

func (d *AkpInstanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AkpInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state *akptypes.AkpInstance
	tflog.Info(ctx, "Reading an Argo CD Instance")
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = ctxutil.SetClientCredential(ctx, d.akpCli.Cred)

	apiReq := &argocdv1.GetOrganizationInstanceRequest{
		OrganizationId: d.akpCli.OrgId,
		IdType:         idv1.Type_NAME,
		Id:             state.Name.ValueString(),
	}

	tflog.Debug(ctx, fmt.Sprintf("apiReq: %s", apiReq))
	apiResp, err := d.akpCli.Cli.GetOrganizationInstance(ctx, apiReq)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Argo CD instance, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Got Argo CD instances")
	protoInstance := &akptypes.ProtoInstance{Instance: apiResp.GetInstance()}
	state = protoInstance.FromProto()

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

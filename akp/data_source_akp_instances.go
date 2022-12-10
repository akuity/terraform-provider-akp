package akp

import (
	"context"
	"fmt"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
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

func NewAkpInstancesDataSource() datasource.DataSource {
	return &AkpInstancesDataSource{}
}

// AkpInstanceDataSource defines the data source implementation.
type AkpInstancesDataSource struct {
	akpCli *AkpCli
}

type AkpInstancesDataSourceModel struct {
	Id         types.String           `tfsdk:"id"`
	Instances []*akptypes.AkpInstance `tfsdk:"instances"`
}

func (d *AkpInstancesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instances"
}

func (d *AkpInstancesDataSource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "List all Argo CD instances",

		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.StringType,
				Computed:            true,
			},
			"instances": {
				MarkdownDescription: "List of Argo CD instances for organization",
				Computed:            true,
				Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
					"id": {
						MarkdownDescription: "Instance ID",
						Type:                types.StringType,
						Computed:            true,
					},
					"name": {
						MarkdownDescription: "Instance Name",
						Type:                types.StringType,
						Computed:            true,
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
				}),
			},
		},
	}, nil
}

func (d *AkpInstancesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AkpInstancesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AkpInstancesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Reading an Argo CD Instances")

	ctx = ctxutil.SetClientCredential(ctx, d.akpCli.Cred)
	apiResp, err := d.akpCli.Cli.ListOrganizationInstances(ctx, &argocdv1.ListOrganizationInstancesRequest{
		OrganizationId: d.akpCli.OrgId,
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Argo CD instances, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Got Argo CD instances")
	instances := apiResp.GetInstances()
	for _, instance := range instances {
		protoInstance := &akptypes.ProtoInstance{Instance: instance}
		state.Instances = append(state.Instances, protoInstance.FromProto())
	}

	state.Id = types.StringValue(d.akpCli.OrgId)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

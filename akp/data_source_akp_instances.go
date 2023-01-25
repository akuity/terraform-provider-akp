package akp

import (
	"context"
	"fmt"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	ctxutil "github.com/akuity/api-client-go/pkg/utils/context"
	akptypes "github.com/akuity/terraform-provider-akp/akp/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &AkpInstancesDataSource{}

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

func (d *AkpInstancesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List all Argo CD instances",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
			},
			"instances": schema.ListNestedAttribute{
				MarkdownDescription: "List of Argo CD instances for organization",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Instance ID",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Instance Name",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Instance Description",
							Computed:            true,
						},
						"hostname": schema.StringAttribute{
							MarkdownDescription: "Instance Hostname",
							Computed:            true,
						},
						"version": schema.StringAttribute{
							MarkdownDescription: "Argo CD Version",
							Computed:            true,
						},
					},
				},
			},
		},
	}
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
	apiResp, err := d.akpCli.Cli.ListInstances(ctx, &argocdv1.ListInstancesRequest{
		OrganizationId: d.akpCli.OrgId,
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Argo CD instances, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Got Argo CD instances")
	for _, instance := range apiResp.GetInstances() {
		stateInstance := &akptypes.AkpInstance{}
		resp.Diagnostics.Append(stateInstance.UpdateInstance(instance)...)
		state.Instances = append(state.Instances, stateInstance)
	}

	state.Id = types.StringValue(d.akpCli.OrgId)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

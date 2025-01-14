package akp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &AkpKargoAgentsDataSource{}

func NewAkpKargoAgentsDataSource() datasource.DataSource {
	return &AkpKargoAgentsDataSource{}
}

// AkpKargoAgentsDataSource defines the data source implementation.
type AkpKargoAgentsDataSource struct {
	akpCli *AkpCli
}

func (d *AkpKargoAgentsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kargo_agents"
}

func (d *AkpKargoAgentsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AkpKargoAgentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading a Kargo Agents Datasource")
	var data types.KargoAgents

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, d.akpCli.Cred.Scheme(), d.akpCli.Cred.Credential())

	apiReq := &kargov1.ListKargoInstanceAgentsRequest{
		OrganizationId: d.akpCli.OrgId,
		InstanceId:     data.InstanceID.ValueString(),
	}
	tflog.Debug(ctx, fmt.Sprintf("List Kargo agents request: %s", apiReq))
	apiResp, err := d.akpCli.KargoCli.ListKargoInstanceAgents(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance Kargo agents, got error: %s", err))
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("List Kargo agents response: %s", apiResp))

	data.ID = data.InstanceID
	agents := apiResp.GetAgents()
	for _, agent := range agents {
		stateAgent := types.KargoAgent{
			InstanceID: data.InstanceID,
		}
		stateAgent.Update(ctx, &resp.Diagnostics, agent, nil)
		data.Agents = append(data.Agents, stateAgent)
	}
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

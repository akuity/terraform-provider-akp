package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"

	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &AkpInstanceDataSource{}

func NewAkpInstanceDataSource() datasource.DataSource {
	return &AkpInstanceDataSource{}
}

// AkpInstanceDataSource defines the data source implementation.
type AkpInstanceDataSource struct {
	BaseDataSource
}

func (r *AkpInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *AkpInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading an Instance Datasource")
	var data types.InstanceDataSource
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance := &types.Instance{
		Name: data.Name,
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	if err := refreshState(ctx, &resp.Diagnostics, r.akpCli, instance, &argocdv1.GetInstanceRequest{
		OrganizationId: r.akpCli.OrgId,
		IdType:         idv1.Type_NAME,
		Id:             data.Name.ValueString(),
	}, true); err != nil {
		resp.Diagnostics.AddError("Failed to refresh instance state", err.Error())
		return
	}
	managedSecrets, err := readInstanceManagedSecrets(ctx, &resp.Diagnostics, r.akpCli, instance)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance managed secrets", err.Error())
		return
	}
	data = types.NewInstanceDataSourceModel(instance, managedSecrets)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func readInstanceManagedSecrets(ctx context.Context, diagnostics *diag.Diagnostics, cli *AkpCli, instance *types.Instance) (map[string]*types.ManagedSecretDataSource, error) {
	workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, instance.Workspace.ValueString())
	if err != nil {
		return nil, errors.Wrap(err, "unable to get workspace")
	}
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.ListInstanceManagedSecretsResponse, error) {
		return cli.Cli.ListInstanceManagedSecrets(ctx, &argocdv1.ListInstanceManagedSecretsRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    workspace.GetId(),
			InstanceId:     instance.ID.ValueString(),
		})
	}, "ListInstanceManagedSecrets")
	if err != nil {
		return nil, errors.Wrap(err, "unable to list managed secrets")
	}
	return types.ToManagedSecretsDataSourceModel(ctx, diagnostics, resp.GetManagedSecrets()), nil
}

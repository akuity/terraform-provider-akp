package akp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

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

func (r *AkpInstanceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := toDataSourceAttributes(getAKPInstanceAttributes(), "name")
	// These values are never returned. Even null sensitive attributes would make
	// existing outputs of the whole data source require sensitive = true.
	for _, name := range []string{
		"argocd_secret", "application_set_secret", "argocd_notifications_secret",
		"argocd_image_updater_secret", "repo_credential_secrets", "repo_template_credential_secrets",
	} {
		delete(attrs, name)
	}
	instanceSpec := attrs["argocd"].(schema.SingleNestedAttribute).Attributes["spec"].(schema.SingleNestedAttribute).Attributes["instance_spec"].(schema.SingleNestedAttribute)
	delete(instanceSpec.Attributes, "metrics_ingress_password_hash")
	// Managed secrets are shaped differently here: the resource's data is write-only
	// and never returned, while the API does report the key names.
	attrs["managed_secrets"] = schema.MapNestedAttribute{
		MarkdownDescription: "Managed secrets on the instance. Secret values are not returned.",
		Computed:            true,
		NestedObject:        schema.NestedAttributeObject{Attributes: getManagedSecretDataSourceAttributes()},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets information about an Argo CD instance by its name",
		Attributes:          attrs,
	}
}

func getManagedSecretDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"labels": schema.MapAttribute{
			MarkdownDescription: "Additional labels set on the secret.",
			Computed:            true,
			ElementType:         tftypes.StringType,
		},
		"allowed_clusters": schema.ListAttribute{
			MarkdownDescription: "Names of managed clusters the secret is synced to.",
			Computed:            true,
			ElementType:         tftypes.StringType,
		},
		"cluster_selector": schema.StringAttribute{
			MarkdownDescription: "Kubernetes label selector that selects the managed clusters the secret is synced to.",
			Computed:            true,
		},
		"secret_keys": schema.ListAttribute{
			MarkdownDescription: "Names of keys stored in the secret. Secret values are not returned.",
			Computed:            true,
			ElementType:         tftypes.StringType,
		},
	}
}

func (r *AkpInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading an Instance Datasource")
	var data types.InstanceDataSource
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = r.AuthCtx(ctx)

	instance := &types.Instance{Name: data.Name}
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
		return nil, fmt.Errorf("unable to get workspace: %w", err)
	}
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.ListInstanceManagedSecretsResponse, error) {
		return cli.Cli.ListInstanceManagedSecrets(ctx, &argocdv1.ListInstanceManagedSecretsRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    workspace.GetId(),
			InstanceId:     instance.ID.ValueString(),
		})
	}, "ListInstanceManagedSecrets")
	if err != nil {
		return nil, fmt.Errorf("unable to list managed secrets: %w", err)
	}
	return types.ToManagedSecretsDataSourceModel(ctx, diagnostics, resp.GetManagedSecrets()), nil
}

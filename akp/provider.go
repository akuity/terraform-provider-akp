package akp

import (
	"context"
	"fmt"
	"os"

	"github.com/akuity/api-client-go/pkg/api/gateway/accesscontrol"
	gwoption "github.com/akuity/api-client-go/pkg/api/gateway/option"
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = &AkpProvider{}

type AkpProvider struct {
	version string
}

type AkpProviderModel struct {
	ServerUrl        types.String `tfsdk:"server_url"`
	ApiKeyId         types.String `tfsdk:"api_key_id"`
	ApiKeySecret     types.String `tfsdk:"api_key_secret"`
	OrganizationName types.String `tfsdk:"org_name"`
	SkipTLSVerify    types.Bool   `tfsdk:"skip_tls_verify"`
}

type AkpCli struct {
	Cli   argocdv1.ArgoCDServiceGatewayClient
	Cred  accesscontrol.ClientCredential
	OrgId string
}

func (p *AkpProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "akp"
	resp.Version = p.version
}

func (p *AkpProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"server_url": schema.StringAttribute{
				MarkdownDescription: "Akuity Platform API URL, default: `https://akuity.cloud`. You can use environment variable `AKUITY_SERVER_URL` instead",
				Optional:            true,
			},
			"skip_tls_verify": schema.BoolAttribute{
				MarkdownDescription: "Skip TLS Verify. Only use for testing self-hosted version",
				Optional:            true,
			},
			"org_name": schema.StringAttribute{
				MarkdownDescription: "Organization Name",
				Required:            true,
			},
			"api_key_id": schema.StringAttribute{
				MarkdownDescription: "API Key Id. Use environment variable `AKUITY_API_KEY_ID`",
				Optional:            true,
				Sensitive:           true,
			},
			"api_key_secret": schema.StringAttribute{
				MarkdownDescription: "API Key Secret, Use environment variable `AKUITY_API_KEY_SECRET`",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *AkpProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Akuity Provider")

	var config AkpProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ServerUrl := os.Getenv("AKUITY_SERVER_URL")
	apiKeyID := os.Getenv("AKUITY_API_KEY_ID")
	apiKeySecret := os.Getenv("AKUITY_API_KEY_SECRET")

	skipTLSVerify := config.SkipTLSVerify.ValueBool()
	if ServerUrl == "" {
		ServerUrl = config.ServerUrl.ValueString()
	}
	if apiKeyID == "" {
		apiKeyID = config.ApiKeyId.ValueString()
	}
	if apiKeySecret == "" {
		apiKeySecret = config.ApiKeySecret.ValueString()
	}
	if ServerUrl == "" {
		ServerUrl = "https://akuity.cloud"
	}
	if apiKeyID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key_id"),
			"Missing Akuity Platform API Key Id",
			"The provider cannot create the Akuity Platform API client as there is an missing API key. "+
				"Use the AKUITY_API_KEY_ID environment variable to configure it.",
		)
	}
	if apiKeySecret == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key_secret"),
			"Missing Akuity Platform API Key Secret",
			"The provider cannot create the Akuity Platform API client as there is an missing API key. "+
				"Use the AKUITY_API_KEY_SECRET environment variable to configure it.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}
	orgName := config.OrganizationName.ValueString()
	ctx = tflog.SetField(ctx, "server_url", ServerUrl)
	ctx = tflog.SetField(ctx, "skip_tls_verify", skipTLSVerify)
	ctx = tflog.SetField(ctx, "api_key_id", apiKeyID)
	ctx = tflog.SetField(ctx, "org_name", orgName)

	tflog.Debug(ctx, "Getting Organization ID by name")

	cred := accesscontrol.NewAPIKeyCredential(apiKeyID, apiKeySecret)
	// Get Organizaton ID by name
	ctx = httpctx.SetAuthorizationHeader(ctx, cred.Scheme(), cred.Credential())
	gwc := gwoption.NewClient(ServerUrl, skipTLSVerify)
	orgc := orgcv1.NewOrganizationServiceGatewayClient(gwc)
	res, err := orgc.GetOrganization(ctx, &orgcv1.GetOrganizationRequest{
		Id:     orgName,
		IdType: idv1.Type_NAME,
	})
	tflog.Debug(ctx, fmt.Sprintf("Res: %s", res))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Akuity Platform API Client",
			"An unexpected error occurred when creating the Akuity Platform API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Akuity Platform Client Error: "+err.Error(),
		)
		return
	}

	orgID := res.Organization.Id
	tflog.Info(ctx, "Connection successful", map[string]any{"org_id": orgID})

	argoc := argocdv1.NewArgoCDServiceGatewayClient(gwc)
	akpCli := &AkpCli{
		Cli:   argoc,
		Cred:  cred,
		OrgId: orgID,
	}
	resp.DataSourceData = akpCli
	resp.ResourceData = akpCli
}

func (p *AkpProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAkpInstanceResource,
		NewAkpClusterResource,
	}
}

func (p *AkpProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAkpInstanceDataSource,
		NewAkpClusterDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AkpProvider{
			version: version,
		}
	}
}

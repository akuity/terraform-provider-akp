package akp

import (
	"context"
	"fmt"
	"os"

	"github.com/akuity/api-client-go/pkg/api/gateway/accesscontrol"
	gwclient "github.com/akuity/api-client-go/pkg/api/gateway/client"
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"

	"github.com/akuity/api-client-go/pkg/api/client"
	ctxutil "github.com/akuity/api-client-go/pkg/utils/context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = &AkpProvider{}
var _ provider.ProviderWithMetadata = &AkpProvider{}

type AkpProvider struct {
	version string
}

type AkpProviderModel struct {
	ApiHost          types.String `tfsdk:"api_host"`
	ApiKeyId         types.String `tfsdk:"api_key_id"`
	ApiKeySecret     types.String `tfsdk:"api_key_secret"`
	OrganizationName types.String `tfsdk:"org_name"`
	SkipTLSVerify    types.Bool   `tfsdk:"skip_tls_verify"`
}

type AkpCli struct {
	Cli   client.ArgoCDV1Client
	Cred  accesscontrol.ClientCredential
	OrgId string
}

func (p *AkpProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "akp"
	resp.Version = p.version
}

func (p *AkpProvider) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"api_host": {
				MarkdownDescription: "Akuity Platform API host, default: `https://akuity.cloud`. You can use environment variable `AKUITY_API_HOST` instead",
				Optional:            true,
				Type:                types.StringType,
			},
			"skip_tls_verify": {
				MarkdownDescription: "Skip TLS Verify. Only use for testing self-hosted version",
				Optional:            true,
				Type:                types.BoolType,
			},
			"org_name": {
				MarkdownDescription: "Organization Name",
				Required:            true,
				Type:                types.StringType,
			},
			"api_key_id": {
				MarkdownDescription: "API Key Id. Use environment variable `AKUITY_API_KEY_ID` instead",
				Optional:            true,
				Sensitive:           true,
				Type:                types.StringType,
			},
			"api_key_secret": {
				MarkdownDescription: "API Key Secret, Use environment variable `AKUITY_API_KEY_SECRET` instead",
				Optional:            true,
				Sensitive:           true,
				Type:                types.StringType,
			},
		},
	}, nil
}

func (p *AkpProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Akuity Provider")

	var config AkpProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiHost := os.Getenv("AKUITY_API_HOST")
	apiKeyID := os.Getenv("AKUITY_API_KEY_ID")
	apiKeySecret := os.Getenv("AKUITY_API_KEY_SECRET")

	skipTLSVerify := config.SkipTLSVerify.ValueBool()
	if apiHost == "" {
		apiHost = config.ApiHost.ValueString()
	}
	if apiKeyID == "" {
		apiKeyID = config.ApiKeyId.ValueString()
	}
	if apiKeySecret == "" {
		apiKeySecret = config.ApiKeySecret.ValueString()
	}
	if apiHost == "" {
		apiHost = "https://akuity.cloud"
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
	ctx = tflog.SetField(ctx, "api_host", apiHost)
	ctx = tflog.SetField(ctx, "skip_tls_verify", skipTLSVerify)
	ctx = tflog.SetField(ctx, "api_key_id", apiKeyID)
	ctx = tflog.SetField(ctx, "org_name", orgName)

	tflog.Debug(ctx, "Getting Organization ID by name")

	cred := accesscontrol.NewAPIKeyCredential(apiKeyID, apiKeySecret)
	// Get Organizaton ID by name
	ctx = ctxutil.SetClientCredential(ctx, cred)
	orgc := client.NewOrganizationV1Client(apiHost, gwclient.SkipTLSVerify(skipTLSVerify))
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

	argoc := client.NewArgoCDV1Client(apiHost,
		gwclient.SkipTLSVerify(skipTLSVerify),
	)

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
		NewAkpInstancesDataSource,
		NewAkpInstanceDataSource,
		NewAkpClustersDataSource,
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

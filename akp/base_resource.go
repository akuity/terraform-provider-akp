package akp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
)

// BaseResource provides shared Configure and auth context injection for all resources.
// Embed this in resource structs to eliminate Configure boilerplate.
type BaseResource struct {
	akpCli *AkpCli
}

func (b *BaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	b.akpCli = akpCli
}

func (b *BaseResource) AuthCtx(ctx context.Context) context.Context {
	return httpctx.SetAuthorizationHeader(ctx, b.akpCli.Cred.Scheme(), b.akpCli.Cred.Credential())
}

type BaseDataSource struct {
	akpCli *AkpCli
}

func (b *BaseDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	akpCli, ok := req.ProviderData.(*AkpCli)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *AkpCli, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	b.akpCli = akpCli
}

func (b *BaseDataSource) AuthCtx(ctx context.Context) context.Context {
	return httpctx.SetAuthorizationHeader(ctx, b.akpCli.Cred.Scheme(), b.akpCli.Cred.Credential())
}

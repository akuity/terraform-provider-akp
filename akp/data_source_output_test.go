//go:build !acc

package akp

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDataSourceWholeObjectOutput(t *testing.T) {
	t.Setenv("TF_CLI_CONFIG_FILE", os.DevNull)
	for _, name := range []string{"instance", "kargo_instance"} {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"akp": providerserver.NewProtocol6WithError(&outputTestProvider{Provider: New("test")()}),
				},
				Steps: []resource.TestStep{{
					Config: fmt.Sprintf(`
provider "akp" { org_name = "test" }
data "akp_%[1]s" "test" { name = "example" }
output "instance" { value = data.akp_%[1]s.test }
`, name),
					Check: resource.TestCheckResourceAttr("data.akp_"+name+".test", "name", "example"),
				}},
			})
		})
	}
}

// Keep the real schemas so Terraform Core decides whether a whole-object output
// is sensitive, even when the data source returns null for every secret.
type outputTestProvider struct {
	provider.Provider
}

func (*outputTestProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}

func (*outputTestProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return &outputTestDataSource{NewAkpInstanceDataSource()} },
		func() datasource.DataSource { return &outputTestDataSource{NewAkpKargoDataSource()} },
	}
}

type outputTestDataSource struct {
	datasource.DataSource
}

func (*outputTestDataSource) Read(_ context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	resp.State.Raw = req.Config.Raw
}

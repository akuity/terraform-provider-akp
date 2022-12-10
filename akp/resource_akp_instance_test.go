package akp

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccInstanceResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccInstanceResourceConfig("test one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", "new-instance"),
					resource.TestCheckResourceAttr("akp_instance.test", "description", "test one"),
					resource.TestCheckResourceAttr("akp_instance.test", "version", "v2.5.3"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "hostname"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "akp_instance.test",
				ImportState:       true,
				ImportStateVerify: true,
				// This is not normally necessary, but is here because this
				// example code does not have an actual upstream service.
				// Once the Read method is able to refresh information from
				// the upstream service, this can be removed.
				ImportStateVerifyIgnore: []string{"configurable_attribute"},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccInstanceResourceConfig("test two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "name", "new-instance"),
					resource.TestCheckResourceAttr("akp_instance.test", "description", "test two"),
					resource.TestCheckResourceAttr("akp_instance.test", "version", "v2.5.3"),
					resource.TestCheckResourceAttrSet("akp_instance.test", "hostname"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccInstanceResourceConfig(description string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = "new-instance"
  version = "v2.5.3"
  description = %q
}
`, description)
}

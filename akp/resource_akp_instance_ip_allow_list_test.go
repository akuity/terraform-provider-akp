package akp

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
)

func TestAccInstanceIPAllowListResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - single entry
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfig(
					getInstanceId(),
					[]map[string]string{
						{"ip": "192.168.1.0/24", "description": "Office network"},
					},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_instance_ip_allow_list.test", "id"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "instance_id", getInstanceId()),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.#", "1"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.0.ip", "192.168.1.0/24"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.0.description", "Office network"),
					testAccCheckInstanceIPAllowListExists("akp_instance_ip_allow_list.test", "192.168.1.0/24"),
				),
			},
			// ImportState testing
			// Note: We don't use ImportStateVerify because the resource ID is a generated UUID
			// that will be different after import. The import functionality is tested by
			// verifying the subsequent update step works correctly with the imported state.
			{
				ResourceName:  "akp_instance_ip_allow_list.test",
				ImportState:   true,
				ImportStateId: getInstanceId(),
			},
			// Update - add more entries to the same resource
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfig(
					getInstanceId(),
					[]map[string]string{
						{"ip": "192.168.1.0/24", "description": "Office network"},
						{"ip": "10.0.0.0/8", "description": "Internal network"},
					},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.#", "2"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.0.ip", "192.168.1.0/24"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.1.ip", "10.0.0.0/8"),
					testAccCheckInstanceIPAllowListExists("akp_instance_ip_allow_list.test", "192.168.1.0/24"),
					testAccCheckInstanceIPAllowListExists("akp_instance_ip_allow_list.test", "10.0.0.0/8"),
				),
			},
			// Update - modify description
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfig(
					getInstanceId(),
					[]map[string]string{
						{"ip": "192.168.1.0/24", "description": "Updated office network"},
						{"ip": "10.0.0.0/8", "description": "Updated internal network"},
					},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.0.description", "Updated office network"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.1.description", "Updated internal network"),
				),
			},
			// Update - remove one entry
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfig(
					getInstanceId(),
					[]map[string]string{
						{"ip": "192.168.1.0/24", "description": "Updated office network"},
					},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.#", "1"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.0.ip", "192.168.1.0/24"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccInstanceIPAllowListResource_MultipleResources(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckInstanceIPAllowListDestroy,
		Steps: []resource.TestStep{
			// Create multiple resources, each managing different IPs
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfigMultiple(getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check office resource
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.office", "entries.#", "2"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.office", "entries.0.ip", "10.0.0.0/8"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.office", "entries.0.description", "Office network"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.office", "entries.1.ip", "192.168.1.0/24"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.office", "entries.1.description", "Office WiFi"),
					// Check vpn resource
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.vpn", "entries.#", "1"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.vpn", "entries.0.ip", "172.16.0.0/12"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.vpn", "entries.0.description", "VPN network"),
					// Check home resource
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.home", "entries.#", "1"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.home", "entries.0.ip", "203.0.113.42/32"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.home", "entries.0.description", "Home IP"),
					// Verify all IPs exist in the instance
					testAccCheckInstanceIPAllowListExists("akp_instance_ip_allow_list.office", "10.0.0.0/8"),
					testAccCheckInstanceIPAllowListExists("akp_instance_ip_allow_list.office", "192.168.1.0/24"),
					testAccCheckInstanceIPAllowListExists("akp_instance_ip_allow_list.vpn", "172.16.0.0/12"),
					testAccCheckInstanceIPAllowListExists("akp_instance_ip_allow_list.home", "203.0.113.42/32"),
				),
			},
			// Update one resource, others should remain unchanged
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfigMultipleUpdated(getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Office resource was updated - added an IP
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.office", "entries.#", "3"),
					// VPN and home should remain the same
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.vpn", "entries.#", "1"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.home", "entries.#", "1"),
				),
			},
		},
	})
}

func TestAccInstanceIPAllowListResource_DuplicateIP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create first resource with an IP
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfig(
					getInstanceId(),
					[]map[string]string{
						{"ip": "198.51.100.0/24", "description": "Test network"},
					},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.0.ip", "198.51.100.0/24"),
				),
			},
			// Try to create duplicate in another resource - should fail
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfig(
					getInstanceId(),
					[]map[string]string{
						{"ip": "198.51.100.0/24", "description": "Test network"},
					},
				) + testAccInstanceIPAllowListResourceConfigDuplicate(
					getInstanceId(),
					"198.51.100.0/24",
					"Duplicate network",
				),
				ExpectError: regexp.MustCompile("already exists in the allow list"),
			},
		},
	})
}

func TestAccInstanceIPAllowListResource_DuplicateInSameResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Try to create with duplicate IPs in the same resource - should fail
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfig(
					getInstanceId(),
					[]map[string]string{
						{"ip": "198.51.100.0/24", "description": "First"},
						{"ip": "198.51.100.0/24", "description": "Duplicate"},
					},
				),
				ExpectError: regexp.MustCompile("appears multiple times"),
			},
		},
	})
}

func TestAccInstanceIPAllowListResource_IPv6(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create IPv6 entry
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfig(
					getInstanceId(),
					[]map[string]string{
						{"ip": "2001:db8::/32", "description": "IPv6 network"},
					},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.0.ip", "2001:db8::/32"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.0.description", "IPv6 network"),
					testAccCheckInstanceIPAllowListExists("akp_instance_ip_allow_list.test", "2001:db8::/32"),
				),
			},
		},
	})
}

func TestAccInstanceIPAllowListResource_NoDescription(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without description
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfigNoDescription(
					getInstanceId(),
					"192.0.2.0/24",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.0.ip", "192.0.2.0/24"),
					testAccCheckInstanceIPAllowListExists("akp_instance_ip_allow_list.test", "192.0.2.0/24"),
				),
			},
			// Add description via update
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfig(
					getInstanceId(),
					[]map[string]string{
						{"ip": "192.0.2.0/24", "description": "Added description"},
					},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.0.description", "Added description"),
				),
			},
		},
	})
}

func TestAccInstanceIPAllowListResource_Migration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Start with instance managing IP allow list
			{
				Config: providerConfig + testAccInstanceWithIPAllowList(
					getInstanceId(),
					"10.0.0.0/8",
					"Original entry",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.ip_allow_list.0.ip", "10.0.0.0/8"),
				),
			},
			// Migrate to separate resource by removing from instance and adding dedicated resource
			// The instance resource will now ignore ip_allow_list changes since it's not in the config
			{
				Config: providerConfig + testAccInstanceWithoutIPAllowList(getInstanceId()) +
					testAccInstanceIPAllowListResourceConfig(
						getInstanceId(),
						[]map[string]string{
							{"ip": "10.0.0.0/8", "description": "Migrated entry"},
						},
					),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckInstanceIPAllowListExists("akp_instance_ip_allow_list.test", "10.0.0.0/8"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.0.ip", "10.0.0.0/8"),
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.0.description", "Migrated entry"),
					// Verify instance resource still sees the IP in state (it's preserved)
					resource.TestCheckResourceAttr("akp_instance.test", "argocd.spec.instance_spec.ip_allow_list.0.ip", "10.0.0.0/8"),
				),
			},
			// Add more entries via the dedicated resource
			{
				Config: providerConfig + testAccInstanceWithoutIPAllowList(getInstanceId()) +
					testAccInstanceIPAllowListResourceConfig(
						getInstanceId(),
						[]map[string]string{
							{"ip": "10.0.0.0/8", "description": "Migrated entry"},
							{"ip": "192.168.1.0/24", "description": "Additional entry"},
						},
					),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.#", "2"),
					testAccCheckInstanceIPAllowListExists("akp_instance_ip_allow_list.test", "10.0.0.0/8"),
					testAccCheckInstanceIPAllowListExists("akp_instance_ip_allow_list.test", "192.168.1.0/24"),
				),
			},
		},
	})
}

// testAccCheckInstanceIPAllowListDestroy verifies that all IP allow list resources have been destroyed
func testAccCheckInstanceIPAllowListDestroy(s *terraform.State) error {
	// Get the instance ID from any of the destroyed resources
	var instanceID string
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "akp_instance_ip_allow_list" {
			instanceID = rs.Primary.Attributes["instance_id"]
			break
		}
	}

	if instanceID == "" {
		// No IP allow list resources were created, nothing to check
		return nil
	}

	// Get the test client
	akpCli := getTestAkpCli()
	if akpCli == nil {
		return fmt.Errorf("could not get test client")
	}

	ctx := context.Background()
	ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())

	// Get the instance to check the IP allow list
	// Note: We use GetInstance instead of ExportInstance because GetInstance is much faster
	// and doesn't require connecting to the k3s control plane
	getResp, err := akpCli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
		OrganizationId: akpCli.OrgId,
		Id:             instanceID,
		IdType:         idv1.Type_ID,
	})
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Check if the IP allow list is empty
	ipAllowList := getResp.Instance.GetSpec().GetIpAllowList()
	if len(ipAllowList) > 0 {
		// List the remaining IPs for debugging
		var remainingIPs []string
		for _, entry := range ipAllowList {
			remainingIPs = append(remainingIPs, entry.Ip)
		}
		return fmt.Errorf("IP allow list still contains %d entries after destroy: %v", len(ipAllowList), remainingIPs)
	}

	return nil
}

// testAccCheckInstanceIPAllowListExists checks if a specific IP exists in the instance's allow list
func testAccCheckInstanceIPAllowListExists(resourceName, expectedIP string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		instanceID := rs.Primary.Attributes["instance_id"]
		if instanceID == "" {
			return fmt.Errorf("instance_id is not set")
		}

		// Get the test client
		akpCli := getTestAkpCli()
		if akpCli == nil {
			return fmt.Errorf("could not get test client")
		}

		ctx := context.Background()
		ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())

		// Get the instance to check the IP allow list
		// Note: We use GetInstance instead of ExportInstance because GetInstance is much faster
		// and doesn't require connecting to the k3s control plane
		getResp, err := akpCli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
			OrganizationId: akpCli.OrgId,
			Id:             instanceID,
			IdType:         idv1.Type_ID,
		})
		if err != nil {
			return fmt.Errorf("failed to get instance: %w", err)
		}

		// Check if the IP exists in the allow list
		ipAllowList := getResp.Instance.GetSpec().GetIpAllowList()
		if ipAllowList == nil {
			return fmt.Errorf("ipAllowList is nil or empty")
		}

		found := false
		for _, entry := range ipAllowList {
			if entry.Ip == expectedIP {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("IP %s not found in instance %s allow list", expectedIP, getResp.Instance.Id)
		}

		return nil
	}
}

// Config helper functions

func testAccInstanceIPAllowListResourceConfig(instanceID string, entries []map[string]string) string {
	entriesStr := ""
	for _, entry := range entries {
		ip := entry["ip"]
		desc := entry["description"]
		entriesStr += fmt.Sprintf(`
    {
      ip          = %q
      description = %q
    },`, ip, desc)
	}

	return fmt.Sprintf(`
resource "akp_instance_ip_allow_list" "test" {
  instance_id = %q
  entries = [%s
  ]
}
`, instanceID, entriesStr)
}

func testAccInstanceIPAllowListResourceConfigNoDescription(instanceID, ip string) string {
	return fmt.Sprintf(`
resource "akp_instance_ip_allow_list" "test" {
  instance_id = %q
  entries = [
    {
      ip = %q
    }
  ]
}
`, instanceID, ip)
}

func testAccInstanceIPAllowListResourceConfigMultiple(instanceID string) string {
	return fmt.Sprintf(`
resource "akp_instance_ip_allow_list" "office" {
  instance_id = %[1]q
  entries = [
    {
      ip          = "10.0.0.0/8"
      description = "Office network"
    },
    {
      ip          = "192.168.1.0/24"
      description = "Office WiFi"
    }
  ]
}

resource "akp_instance_ip_allow_list" "vpn" {
  instance_id = %[1]q
  entries = [
    {
      ip          = "172.16.0.0/12"
      description = "VPN network"
    }
  ]
}

resource "akp_instance_ip_allow_list" "home" {
  instance_id = %[1]q
  entries = [
    {
      ip          = "203.0.113.42/32"
      description = "Home IP"
    }
  ]
}
`, instanceID)
}

func testAccInstanceIPAllowListResourceConfigMultipleUpdated(instanceID string) string {
	return fmt.Sprintf(`
resource "akp_instance_ip_allow_list" "office" {
  instance_id = %[1]q
  entries = [
    {
      ip          = "10.0.0.0/8"
      description = "Office network"
    },
    {
      ip          = "192.168.1.0/24"
      description = "Office WiFi"
    },
    {
      ip          = "192.168.2.0/24"
      description = "Office Guest WiFi"
    }
  ]
}

resource "akp_instance_ip_allow_list" "vpn" {
  instance_id = %[1]q
  entries = [
    {
      ip          = "172.16.0.0/12"
      description = "VPN network"
    }
  ]
}

resource "akp_instance_ip_allow_list" "home" {
  instance_id = %[1]q
  entries = [
    {
      ip          = "203.0.113.42/32"
      description = "Home IP"
    }
  ]
}
`, instanceID)
}

func testAccInstanceIPAllowListResourceConfigDuplicate(instanceID, ip, description string) string {
	return fmt.Sprintf(`
resource "akp_instance_ip_allow_list" "duplicate" {
  instance_id = %[1]q
  entries = [
    {
      ip          = %[2]q
      description = %[3]q
    }
  ]
}
`, instanceID, ip, description)
}

// TestAccInstanceIPAllowListResource_PreservesInstanceSettings verifies that adding/modifying
// IP allow list entries doesn't modify other instance settings (description, version, etc.)
func TestAccInstanceIPAllowListResource_PreservesInstanceSettings(t *testing.T) {
	instanceID := getInstanceId()

	// Capture the initial instance state before any IP allow list changes
	var initialDescription string
	var initialVersion string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Capture initial instance state
			{
				Config: providerConfig + testAccEmptyConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCaptureInstanceState(instanceID, &initialDescription, &initialVersion),
				),
			},
			// Step 2: Add IP allow list entries
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfig(
					instanceID,
					[]map[string]string{
						{"ip": "10.10.10.0/24", "description": "Test network for preservation test"},
					},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.#", "1"),
					testAccCheckInstanceIPAllowListExists("akp_instance_ip_allow_list.test", "10.10.10.0/24"),
					// Verify instance settings are preserved
					testAccVerifyInstanceSettingsPreserved(instanceID, &initialDescription, &initialVersion),
				),
			},
			// Step 3: Update IP allow list entries (add more)
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfig(
					instanceID,
					[]map[string]string{
						{"ip": "10.10.10.0/24", "description": "Test network for preservation test"},
						{"ip": "10.20.20.0/24", "description": "Second test network"},
					},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.#", "2"),
					// Verify instance settings are still preserved after update
					testAccVerifyInstanceSettingsPreserved(instanceID, &initialDescription, &initialVersion),
				),
			},
			// Step 4: Modify IP allow list entries (change description)
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfig(
					instanceID,
					[]map[string]string{
						{"ip": "10.10.10.0/24", "description": "Updated description"},
						{"ip": "10.20.20.0/24", "description": "Updated second network"},
					},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify instance settings are still preserved after description update
					testAccVerifyInstanceSettingsPreserved(instanceID, &initialDescription, &initialVersion),
				),
			},
			// Step 5: Remove some IP allow list entries
			{
				Config: providerConfig + testAccInstanceIPAllowListResourceConfig(
					instanceID,
					[]map[string]string{
						{"ip": "10.10.10.0/24", "description": "Only remaining entry"},
					},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_instance_ip_allow_list.test", "entries.#", "1"),
					// Verify instance settings are still preserved after removal
					testAccVerifyInstanceSettingsPreserved(instanceID, &initialDescription, &initialVersion),
				),
			},
		},
	})
}

// testAccEmptyConfig returns an empty config to use for capturing initial state
func testAccEmptyConfig() string {
	return ""
}

// testAccCaptureInstanceState captures the current instance description and version
func testAccCaptureInstanceState(instanceID string, description, version *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		akpCli := getTestAkpCli()
		if akpCli == nil {
			return fmt.Errorf("could not get test client")
		}

		ctx := context.Background()
		ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())

		getResp, err := akpCli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
			OrganizationId: akpCli.OrgId,
			Id:             instanceID,
			IdType:         idv1.Type_ID,
		})
		if err != nil {
			return fmt.Errorf("failed to get instance: %w", err)
		}

		*description = getResp.Instance.Description
		*version = getResp.Instance.Version

		return nil
	}
}

// testAccVerifyInstanceSettingsPreserved verifies that the instance description and version
// haven't changed from the initial captured values
func testAccVerifyInstanceSettingsPreserved(instanceID string, initialDescription, initialVersion *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		akpCli := getTestAkpCli()
		if akpCli == nil {
			return fmt.Errorf("could not get test client")
		}

		ctx := context.Background()
		ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())

		getResp, err := akpCli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
			OrganizationId: akpCli.OrgId,
			Id:             instanceID,
			IdType:         idv1.Type_ID,
		})
		if err != nil {
			return fmt.Errorf("failed to get instance: %w", err)
		}

		// Check description is preserved
		if getResp.Instance.Description != *initialDescription {
			return fmt.Errorf("instance description changed: expected %q, got %q",
				*initialDescription, getResp.Instance.Description)
		}

		// Check version is preserved
		if getResp.Instance.Version != *initialVersion {
			return fmt.Errorf("instance version changed: expected %q, got %q",
				*initialVersion, getResp.Instance.Version)
		}

		return nil
	}
}

// Helper config functions for akp_instance resource

func testAccInstanceWithIPAllowList(instanceID, ip, description string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = "test-instance-%[1]s"
  argocd = {
    "spec" = {
      "instance_spec" = {
        "ip_allow_list" = [
          {
            "ip"          = %[2]q
            "description" = %[3]q
          }
        ]
      }
      "version" = "v2.11.4"
    }
  }
}
`, instanceID, ip, description)
}

func testAccInstanceWithoutIPAllowList(instanceID string) string {
	return fmt.Sprintf(`
resource "akp_instance" "test" {
  name = "test-instance-%[1]s"
  argocd = {
    "spec" = {
      "instance_spec" = {}
      "version" = "v2.11.4"
    }
  }
}
`, instanceID)
}

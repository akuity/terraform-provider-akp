// Org-scoped API key with a one-year expiry. The generated `secret` is only
// returned on create; capture it via the terraform output or state immediately.
//
// `description`, `permissions`, and `ip_allowlist` are updated in place, so
// editing them leaves `secret` alone. Changing `expire_in_duration` or
// `workspace` replaces the key and issues a new secret.
resource "akp_api_key" "ci" {
  description        = "CI deployments"
  expire_in_duration = "8760h"
  permissions = {
    roles = ["member"]
  }
  // Restrict the key to the CI egress ranges. Omit to leave it unrestricted.
  ip_allowlist = ["203.0.113.0/24", "198.51.100.10/32"]
}

resource "akp_workspace" "platform" {
  name = "platform"
}

// Workspace-scoped API key, bound to the workspace created above. The server
// auto-adds an implicit `organization/member` role; only the workspace-scoped
// roles you declare here appear in state.
resource "akp_api_key" "platform_admin" {
  workspace          = akp_workspace.platform.name
  description        = "Platform admin key"
  expire_in_duration = "30d"
  permissions = {
    roles = ["admin"]
  }
}

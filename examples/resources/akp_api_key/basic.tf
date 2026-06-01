// Org-scoped API key with a one-year expiry. The generated `secret` is only
// returned on create; capture it via the terraform output or state immediately.
resource "akp_api_key" "ci" {
  description        = "CI deployments"
  expire_in_duration = "8760h"
  permissions = {
    roles = ["member"]
  }
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

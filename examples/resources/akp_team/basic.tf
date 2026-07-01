// Organization team. Teams group users and can be granted org-level custom
// roles. They can also be added to workspaces via `akp_workspace_member`.
resource "akp_team" "platform" {
  name        = "platform"
  description = "Platform engineering team"
}

// Team with org-level custom roles attached.
resource "akp_custom_role" "api_key_manager" {
  name        = "api-key-manager"
  description = "Manage org-level API keys"
  policy      = "p, role:api-key-manager, organization/apikeys, *, *"
}

resource "akp_team" "operators" {
  name         = "operators"
  description  = "On-call operators"
  custom_roles = [akp_custom_role.api_key_manager.id]
}

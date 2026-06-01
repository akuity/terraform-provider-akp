// Org-scoped custom role. Each non-empty, non-comment line is either a
// permission rule (`p` directive + 4 fields: subject, object, verb, resource)
// or a grouping (`g, sub, role`). Only objects in the org `validActions` map
// are accepted (see internal/services/accesscontrol/accesscontrol.go).
resource "akp_custom_role" "api_key_manager" {
  name        = "api-key-manager"
  description = "Manage org-level API keys"
  policy      = "p, role:api-key-manager, organization/apikeys, *, *"
}

resource "akp_workspace" "platform" {
  name = "platform"
}

// Workspace-scoped custom role. Scope and the available object set differ
// from the org-scoped variant — workspace policies can only reference
// `workspace/*` objects.
resource "akp_custom_role" "workspace_instance_viewer" {
  workspace   = akp_workspace.platform.name
  name        = "instance-viewer"
  description = "Read-only access to instances in this workspace"
  policy      = "p, role:instance-viewer, workspace/instances, get, *"
}

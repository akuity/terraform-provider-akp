resource "akp_workspace" "platform" {
  name = "platform"
}

// Add a user to the workspace by email, as an admin.
resource "akp_workspace_member" "alice" {
  workspace  = akp_workspace.platform.name
  role       = "admin"
  user_email = "alice@example.com"
}

// Add a whole team to the workspace.
resource "akp_team" "platform" {
  name = "platform"
}

resource "akp_workspace_member" "platform_team" {
  workspace = akp_workspace.platform.name
  role      = "member"
  team_name = akp_team.platform.name
}

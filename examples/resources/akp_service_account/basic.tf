// Org-scoped account a GitHub Actions workflow in acme/app assumes. The
// workflow exchanges its OIDC token at
// POST /api/v1/organizations/<org>/oauth/token with this account's `id`.
resource "akp_service_account" "ci" {
  description = "CI deployments"
  permissions = {
    roles = ["member"]
  }
  oidc_binding = {
    issuer_id = akp_oidc_issuer.github_actions.id
    match = {
      sub = "repo:acme/app:ref:refs/heads/main"
    }
  }
  // Restrict the credential to the CI egress ranges. Omit to leave it unrestricted.
  ip_allowlist = ["203.0.113.0/24"]
}

resource "akp_workspace" "platform" {
  name = "platform"
}

// Workspace-scoped account for a workload on the prod cluster. It holds the
// workspace admin role plus the implicit organization membership; only the
// workspace role appears in state. Extra claims narrow the binding further.
resource "akp_service_account" "platform_deployer" {
  workspace   = akp_workspace.platform.name
  description = "Platform deployer"
  permissions = {
    roles = ["admin"]
  }
  oidc_binding = {
    issuer_id = akp_oidc_issuer.k8s.id
    match = {
      sub                       = "system:serviceaccount:platform:deployer"
      "kubernetes.io.namespace" = "platform"
    }
  }
}

variable "kargo_name" {
  type    = string
  default = "github-sso"
}

variable "kargo_version" {
  description = "A Kargo version available in your organization."
  type        = string
}

variable "kargo_hostname" {
  description = "Custom hostname configured to reach the Kargo instance."
  type        = string
}

variable "github_client_id" {
  description = "GitHub OAuth app client ID. Set its callback URL to https://<kargo_hostname>/dex/callback."
  type        = string
}

variable "github_client_secret" {
  description = "GitHub OAuth app client secret."
  type        = string
  sensitive   = true
}

variable "github_org" {
  description = "GitHub organization whose teams grant access to Kargo."
  type        = string
}

resource "akp_kargo_instance" "github_sso" {
  name = var.kargo_name
  kargo = {
    spec = {
      version = var.kargo_version
      fqdn    = var.kargo_hostname
      oidc_config = {
        enabled     = true
        dex_enabled = true
        dex_config = yamlencode({
          connectors = [{
            type = "github"
            id   = "github"
            name = "GitHub"
            config = {
              clientID     = var.github_client_id
              clientSecret = "$GITHUB_CLIENT_SECRET"
              redirectURI  = "https://${var.kargo_hostname}/dex/callback"
              # Dex loads team membership from this organization for the groups claim.
              orgs          = [{ name = var.github_org }]
              teamNameField = "slug"
            }
          }]
        })
        # Dex reads the secret through the environment variable above.
        dex_config_secret = {
          GITHUB_CLIENT_SECRET = var.github_client_secret
        }
        # Replace these team slugs with teams in your GitHub organization.
        admin_account = {
          claims = {
            groups = { values = ["${var.github_org}:platform-admins"] }
          }
        }
        viewer_account = {
          claims = {
            groups = { values = ["${var.github_org}:developers"] }
          }
        }
      }
      kargo_instance_spec = {}
    }
  }
}

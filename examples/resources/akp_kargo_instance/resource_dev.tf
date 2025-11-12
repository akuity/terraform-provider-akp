resource "akp_kargo_instance" "example" {
  name      = "test"
  workspace = "kargo-workspace"
  kargo_cm = {
    adminAccountEnabled  = "true"
    adminAccountTokenTtl = "24h"
  }
  kargo_secret = {
    adminAccountPasswordHash = "$2a$10$wThs/VVwx5Tbygkk5Rzbv.V8hR8JYYmRdBiGjue9pd0YcEXl7.Kn."
  }
  kargo = {
    spec = {
      description = "test-description"
      version     = "v1.8.0-ak.0"
      // only set one of fqdn and subdomain
      fqdn      = ${local.kargo_url}
      subdomain = ""
      
      # OIDC using dex with GitHub Oauth
      oidc_config = {
        enabled     = true
        dex_enabled = true
        dex_config  = <<-EOF
        connectors:
          - type: github
            id: github
            name: GitHub
            config:
              clientID: ${var.GH_OAUTH_CLIENT_ID_KARGO}
              clientSecret: $GITHUB_CLIENT_SECRET
              redirectURI: https://${local.kargo_url}/api/dex/callback
              #orgs:
              #- name: akuity
              #preferredEmailDomain: "akuity.io"
        EOF
        # Define value for secret referenced above.
        dex_config_secret = {
         GITHUB_CLIENT_SECRET = var.GH_OAUTH_CLIENT_SECRET_KARGO
        }
        admin_account = {
          claims = {
            groups = {
              values = ["orgname:team-name"]
            }
          }
        }
        viewer_account = {
          claims = {
            group = {
              values = ["orgname:team-name-devs"]
            }
          }
        }
      }
    }
  }
}

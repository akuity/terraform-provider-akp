resource "akp_instance" "example" {
  name           = "example-argocd-instance-name"
  version        = "v2.6.0"
  description    = "Some description"
  default_policy = "role:readonly"
  subdomain      = "custom"
  web_terminal = {
    enabled = true
  }
  secrets = {
    sso_secret = {
      value = "secret"
    }
  }
  image_updater = {
    secrets = {
      docker_json = {
        value = "secret"
      }
    }
    registries = {
      docker = {
        prefix      = "docker.io"
        api_url     = "https://registry-1.docker.io"
        credentials = "secret:argocd/argocd-image-updater-secret#docker_json"
      }
    }
  }
  declarative_management_enabled = true
  image_updater_enabled          = true
}

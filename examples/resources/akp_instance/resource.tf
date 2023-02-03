resource "akp_instance" "example" {
  name        = "example-argocd-instance-name"
  version     = "v2.5.6"
  description = "Some description"
  rbac_config = {
    default_policy = "role:readonly"
  }
  config = {
    web_terminal = {
      enabled = true
    }
  }
  spec = {
    declarative_management = true
    subdomain              = "custom"
  }
}

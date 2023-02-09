resource "akp_instance" "example" {
  name           = "example-argocd-instance-name"
  version        = "v2.6.0"
  description    = "Some description"
  default_policy = "role:readonly"
  web_terminal = {
    enabled = true
  }
  declarative_management_enabled = true
  subdomain                      = "custom"
}

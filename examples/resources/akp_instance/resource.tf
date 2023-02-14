resource "akp_instance" "example" {
  name           = "example-argocd-instance-name"
  version        = "v2.6.0"
  description    = "Some description"
  default_policy = "role:readonly"
  subdomain      = "custom"
  web_terminal = {
    enabled = true
  }
  secrets = [{
    name  = "slack_token"
    value = "secret"
  }]
  declarative_management_enabled = true
}

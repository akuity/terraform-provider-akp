resource "akp_instance" "argocd" {
  lifecycle {
    prevent_destroy = true
  }

  name = "argocd"
  argocd = {
    "spec" = {
      "instance_spec" = {
        "declarative_management_enabled" = true
        "application_set_extension" = {
          "enabled" = true
        }
        "extensions" = [
          {
            "id"      = "argo_rollouts"
            "version" = "v0.3.7"
          }
        ]
      }
      "version" = "v2.11.4"
    }
  }
}

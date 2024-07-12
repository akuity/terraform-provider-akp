resource "akp_instance" "argocd" {
  name = "argocd"
  argocd = {
    "spec" = {
      "instance_spec" = {
        "declarative_management_enabled" = true
      }
      "version" = "v2.11.4"
    }
  }
}

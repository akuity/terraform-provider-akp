data "akp_instance" "example" {
  name = "test"
}

output "managed_secrets" {
  value = data.akp_instance.example.managed_secrets
}

# On a large instance, read only the objects you use. Each entry is written
# like an argocd_resources key: <apiVersion>/<kind>/<namespace>/<name>.
data "akp_instance" "team_a" {
  name            = "test"
  resource_filter = ["argoproj.io/v1alpha1/AppProject//team-a"]
}

output "team_a_project" {
  value = jsondecode(data.akp_instance.team_a.argocd_resources["argoproj.io/v1alpha1/AppProject//team-a"])
}

data "akp_kargo_instance" "example" {
  name = "test"
}

# Read only the Kargo objects you use. Each entry is written like a
# kargo_resources key: <apiVersion>/<kind>/<namespace>/<name>.
data "akp_kargo_instance" "dev_stage" {
  name            = "test"
  resource_filter = ["kargo.akuity.io/v1alpha1/Stage/my-project/dev"]
}

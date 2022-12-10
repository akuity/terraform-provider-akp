data "akp_instance" "example" {
  name = "example-argocd-instance-name"
}

resource "akp_cluster" "example" {
  name        = "some-value"
  namespace   = "akuity"
  instance_id = data.akp_instance.example.id
}

data "akp_instance" "example" {
  name = "example-argocd-instance-name"
}

data "akp_cluster" "example" {
  instance_id = data.akp_instance.example.id
  name        = "example-cluster-name"
}

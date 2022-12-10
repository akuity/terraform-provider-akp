data "akp_instance" "example" {
  name = "example-argocd-instance-name"
}

data "akp_clusters" "example" {
  instance_id = data.akp_instance.example.id
}

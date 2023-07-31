data "akp_instance" "example" {
  name = "test"
}

data "akp_clusters" "example" {
  instance_id = data.akp_instance.example.id
}

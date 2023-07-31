data "akp_instance" "example" {
  name = "test"
}

data "akp_cluster" "example" {
  instance_id = data.akp_instance.example.id
  name        = "test"
}

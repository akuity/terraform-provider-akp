data "akp_kargo_instance" "example" {
  name = "test"
}

data "akp_kargo_agents" "examples" {
  instance_id = data.akp_kargo_instance.example.id
}

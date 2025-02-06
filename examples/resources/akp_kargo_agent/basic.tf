resource "akp_kargo_agent" "example-agent" {
  instance_id = akp_kargo_instance.example.id
  name        = "test-agent"
  spec = {
    data = {
      target_version = "0.5.52"
      size           = "small"
    }
  }
}

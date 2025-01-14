resource "akp_kargo_instance" "example" {
  name = "test"
  kargo = {
    spec = {
      version             = "v1.1.1"
      kargo_instance_spec = {}
    }
  }
}
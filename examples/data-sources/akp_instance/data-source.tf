data "akp_instance" "example" {
  name = "test"
}

output "managed_secrets" {
  value = data.akp_instance.example.managed_secrets
}

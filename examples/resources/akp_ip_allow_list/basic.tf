resource "akp_instance" "argocd" {
  name = "argocd"
}

resource "akp_instance_ip_allow_list" "example" {
  instance_id = akp_instance.argocd.id
  entries = [
    {
      ip          = "172.16.0.0/12"
      description = "VPN network"
    },
    {
      ip          = "203.0.113.42/32"
      description = "Home IP"
    }
  ]
}
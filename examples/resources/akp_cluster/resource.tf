data "akp_instance" "example" {
  name = "example-argocd-instance-name"
}

resource "akp_cluster" "example" {
  name        = "some-name"
  namespace   = "akuity"
  size        = "small"
  instance_id = data.akp_instance.example.id
  labels = {
    label_1 = "example-label"
  }
  annotations = {
    ann_1 = "example-annotation"
  }
}

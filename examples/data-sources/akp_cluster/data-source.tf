terraform {
  required_providers {
    akp = {
      source = "akuity/akp"
    }
  }
}

provider "akp" {
  org_name = "test"
}

data "akp_instance" "example" {
  name = "test"
}

data "akp_cluster" "example" {
  instance_id = data.akp_instance.example.id
  name        = "test"
}

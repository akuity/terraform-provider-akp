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

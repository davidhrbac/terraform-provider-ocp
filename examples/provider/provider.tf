terraform {
  required_providers {
    ocp = {
      source  = "davidhrbac/ocp"
      version = "~> 0.1"
    }
  }
}

provider "ocp" {
  endpoint             = "https://ocpportal.int.tieto.com/v2/graphql/"
  token                = var.ocp_token
  insecure_skip_verify = false
}

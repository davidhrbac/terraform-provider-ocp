data "ocp_customer" "example" {
  name = "customer-a"
}

data "ocp_project" "example" {
  name        = "project-a"
  customer_id = data.ocp_customer.example.id
}

data "ocp_tier" "bronze" {
  name = "Bronze"
}

data "ocp_vcenter" "example" {
  customer_id = data.ocp_customer.example.id
  name        = "vcenter-01"
}

resource "ocp_virtual_host_caas" "shadow" {
  region     = "FINLAND"
  vcenter_id = data.ocp_vcenter.example.id
  project_id = data.ocp_project.example.id
  tier_id    = data.ocp_tier.bronze.id
  hostname   = "legacy-vm"
  uuid       = "legacy-vm"
  note       = "inventory-only"
}

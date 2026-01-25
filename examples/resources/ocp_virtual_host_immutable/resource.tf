data "ocp_customer" "example" {
  name = "customer-a"
}

data "ocp_project" "example" {
  name        = "project-a"
  customer_id = data.ocp_customer.example.id
}

data "ocp_network" "default" {
  name        = "net-a"
  customer_id = data.ocp_customer.example.id
}

data "ocp_template" "coreos" {
  name        = "coreos"
  customer_id = data.ocp_customer.example.id
  region      = "FINLAND"
}

data "ocp_tier" "bronze" {
  name = "Bronze"
}

resource "ocp_virtual_host_immutable" "example" {
  region               = "FINLAND"
  customer_id          = data.ocp_customer.example.id
  project_id           = data.ocp_project.example.id
  hostname             = "immutable-vm"
  cpu_count            = 4
  memory_size_gb       = 16
  template_id          = data.ocp_template.coreos.id
  tier_id              = data.ocp_tier.bronze.id
  note                 = "managed-by-terraform"
  ignition_config_data = filebase64("./ignition.json")

  interfaces {
    network_id = data.ocp_network.default.id
    ip_list    = ["10.0.0.10"]
  }
}

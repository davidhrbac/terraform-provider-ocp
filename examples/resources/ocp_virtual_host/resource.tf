data "ocp_customer" "example" {
  name = "customer-a"
}

data "ocp_project" "example" {
  name        = "project-a"
  customer_id = data.ocp_customer.example.id
}

data "ocp_domain" "example" {
  customer_id = data.ocp_customer.example.id
  name        = "example.com"
}

data "ocp_network" "default" {
  name        = "net-a"
  customer_id = data.ocp_customer.example.id
}

data "ocp_template" "base" {
  name        = "ubuntu-22-04"
  customer_id = data.ocp_customer.example.id
  region      = "FINLAND"
}

data "ocp_tier" "bronze" {
  name = "Bronze"
}

data "ocp_data_protection_policy" "default" {
  customer_id = data.ocp_customer.example.id
  project_id  = data.ocp_project.example.id
  note        = "daily"
}

resource "ocp_virtual_host" "example" {
  region                 = "FINLAND"
  customer_id            = data.ocp_customer.example.id
  project_id             = data.ocp_project.example.id
  hostname               = "example-vm"
  domain_id              = data.ocp_domain.example.id
  cpu_count              = 4
  cores_per_socket       = 2
  memory_size_gb         = 16
  tier_id                = data.ocp_tier.bronze.id
  template_id            = data.ocp_template.base.id
  note                   = "managed-by-terraform"
  data_protection_policy = data.ocp_data_protection_policy.default.id

  interfaces {
    network_id     = data.ocp_network.default.id
    auto_assign_ip = true
  }
}

# Terraform Provider for OCP

Terraform provider for managing OCP infrastructure via its GraphQL API.

The provider allows you to provision and manage virtual hosts and related
infrastructure using declarative Terraform configuration.

## Requirements

- Terraform >= 1.5
- Access to the OCP API

## Provider Configuration

```hcl
provider "ocp" {
  endpoint = "https://ocp.example.com/v2/graphql"
  token    = var.ocp_token
  insecure = false
}
```

### Arguments

| Name | Description | Required |
|----|------------|----------|
| `endpoint` | Base URL of the OCP API | Yes |
| `token` | Authentication token for the OCP API | Yes |
| `insecure` | Skip TLS certificate verification (use with caution) | No |

The token can also be provided via the `OCP_TOKEN` environment variable.

## Resources

### `ocp_virtual_host`

Creates and manages a virtual host in OCP.

```hcl
resource "ocp_virtual_host" "example" {
  name        = "example-vm"
  customer_id = data.ocp_customer.example.id
  project_id  = data.ocp_project.example.id
  template_id = data.ocp_template.example.id
  tier_id     = data.ocp_tier.fast.id

  interfaces {
    network_id     = data.ocp_network.default.id
    auto_assign_ip = true
  }
}
```

### Update Behavior

Some updates are intentionally restricted to ensure predictable behavior.
For example, sizing changes and tier changes cannot be applied in a single
Terraform run.

## Data Sources

Data sources are provided to resolve object IDs by name.

Available data sources:

- `ocp_customer`
- `ocp_project`
- `ocp_template`
- `ocp_tier`
- `ocp_domain`
- `ocp_network`
- `ocp_data_protection_policy`

Example:

```hcl
data "ocp_customer" "example" {
  name = "customer-a"
}
```

## Error Handling

Errors follow Terraform conventions and are designed to be clear and actionable.

Example:

```
failed to create virtual host: validation error
```

## License

MPL-2.0

package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ocpclient "github.com/davidhrbac/terraform-provider-ocp/internal/client"
)

// ResourceVirtualHost defines the ocp_virtual_host resource schema and CRUD operations.
func ResourceVirtualHost() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceVirtualHostCreate,
		ReadContext:   ResourceVirtualHostRead,
		UpdateContext: ResourceVirtualHostUpdate,
		DeleteContext: ResourceVirtualHostDelete,

		// Import supports: terraform import ocp_virtual_host.<name> <VirtualHost GlobalID>
		// The ID must be the GraphQL GlobalID (VirtualHostNode.id) because Read() uses:
		//   virtualHost(id: GlobalID!)
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Description: "Region.",
				Required:    true,
			},
			"customer_id": {
				Type:        schema.TypeString,
				Description: "ID of the customer that owns the virtual host.",
				Required:    true,
				ForceNew:    true,
			},
			"project_id": {
				Type:        schema.TypeString,
				Description: "ID of the project in which the virtual host is created.",
				Required:    true,
				ForceNew:    true,
			},
			"hostname": {
				Type:        schema.TypeString,
				Description: "Hostname.",
				Required:    true,
			},
			"domain_id": {
				Type:        schema.TypeString,
				Description: "ID of the domain.",
				Required:    true,
			},
			"allow_resize_restart": {
				Type:        schema.TypeBool,
				Description: "Allow resize restart.",
				Optional:    true,
				Default:     true,
			},
			"cpu_count": {
				Type:        schema.TypeInt,
				Description: "Cpu count.",
				Required:    true,
			},
			"cores_per_socket": {
				Type:        schema.TypeInt,
				Description: "Cores per socket.",
				Optional:    true,
				Default:     1,
			},
			"memory_size_gb": {
				Type:        schema.TypeInt,
				Description: "Memory size gb.",
				Required:    true,
			},
			"tier_id": {
				Type:        schema.TypeString,
				Description: "ID of the storage tier assigned to the virtual host.",
				Required:    true,
			},
			"template_id": {
				Type:        schema.TypeString,
				Description: "ID of the template used to create the virtual host.",
				Required:    true,
				ForceNew:    true,
			},
			"note": {
				Type:        schema.TypeString,
				Description: "Note.",
				Required:    true,
			},
			"data_protection_policy": {
				Type:        schema.TypeString,
				Description: "Data protection policy.",
				Required:    true,
			},
			"interfaces": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_id": {
							Type:        schema.TypeString,
							Description: "ID of the network for this interface.",
							Required:    true,
						},
						"auto_assign_ip": {
							Type:        schema.TypeBool,
							Description: "Whether the IP should be assigned automatically.",
							Optional:    true,
							Default:     true,
						},
						"ip": {
							Type:        schema.TypeString,
							Description: "IP address for this interface.",
							Optional:    true,
						},
					},
				},
			},
			"uuid": {
				Type:        schema.TypeString,
				Description: "Uuid.",
				Computed:    true,
			},
			"status": {
				Type:        schema.TypeString,
				Description: "Status.",
				Computed:    true,
			},
		},
	}
}

const mutationCreateVM = `
mutation CreateVm($input: VirtualHostCreateInput!) {
  virtualHostCreate(input: $input) {
    __typename
    ... on VirtualHostCreated {
      virtualHost {
        id
        uuid
        hostname
        state
        cpuCount
        coresPerSocket
        memorySizeMB
        tier { id }
        domain { id }
        template { id }
        project { id }
        customer { id }
        region
      }
    }
    ... on ValidationErrors {
      message
      errors {
        field
        messages
      }
    }
    ... on Unauthorized {
      message
    }
    ... on OperationUnavailable {
      message
      reasons
    }
  }
}
`

// ResourceVirtualHostCreate creates a new virtual host via the API.
func ResourceVirtualHostCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	// interfaces -> TemplateInterfaceInput[]
	rawIfaces := d.Get("interfaces").([]interface{})
	if len(rawIfaces) == 0 {
		return diag.Errorf("at least 1 interfaces block is required")
	}
	var ifaces []map[string]interface{}
	for _, raw := range rawIfaces {
		m := raw.(map[string]interface{})

		iface := map[string]interface{}{
			"network": m["network_id"].(string),
		}

		// autoAssignIp
		if v, ok := m["auto_assign_ip"]; ok {
			iface["autoAssignIp"] = v.(bool)
		}

		// ip -> ipList (single element)
		if ipRaw, ok := m["ip"]; ok {
			ip := ipRaw.(string)
			if ip != "" {
				iface["ipList"] = []string{ip}
			}
		}

		ifaces = append(ifaces, iface)
	}

	// Build VirtualHostCreateInput according to the API schema expected by virtualHostCreate.
	input := map[string]interface{}{
		"region":               d.Get("region").(string),
		"customer":             d.Get("customer_id").(string),
		"project":              d.Get("project_id").(string),
		"hostname":             d.Get("hostname").(string),
		"domain":               d.Get("domain_id").(string),
		"cpuCount":             d.Get("cpu_count").(int),
		"coresPerSocket":       d.Get("cores_per_socket").(int),
		"memorySizeGB":         d.Get("memory_size_gb").(int),
		"tier":                 d.Get("tier_id").(string),
		"template":             d.Get("template_id").(string),
		"note":                 d.Get("note").(string),
		"dataProtectionPolicy": d.Get("data_protection_policy").(string),
		"interfaceList":        ifaces,
	}

	vars := map[string]interface{}{
		"input": input,
	}

	// Create returns a union payload; on success we use the returned virtualHost directly (no follow-up listing).
	type virtualHost struct {
		ID             string              `json:"id"`
		UUID           string              `json:"uuid"`
		Hostname       string              `json:"hostname"`
		State          string              `json:"state"`
		CpuCount       int                 `json:"cpuCount"`
		CoresPerSocket int                 `json:"coresPerSocket"`
		MemorySizeMB   int                 `json:"memorySizeMB"`
		Tier           struct{ ID string } `json:"tier"`
		Domain         struct{ ID string } `json:"domain"`
		Template       struct{ ID string } `json:"template"`
		Project        struct{ ID string } `json:"project"`
		Customer       struct{ ID string } `json:"customer"`
		Region         string              `json:"region"`
	}

	type fieldMessages struct {
		Field    string   `json:"field"`
		Messages []string `json:"messages"`
	}

	type virtualHostCreatePayload struct {
		Typename    string          `json:"__typename"`
		VirtualHost *virtualHost    `json:"virtualHost,omitempty"`
		Message     string          `json:"message,omitempty"`
		Errors      []fieldMessages `json:"errors,omitempty"`
		Reasons     []string        `json:"reasons,omitempty"`
	}

	var createResp struct {
		VirtualHostCreate virtualHostCreatePayload `json:"virtualHostCreate"`
	}

	if err := client.Do(mutationCreateVM, vars, &createResp); err != nil {
		return diag.FromErr(err)
	}

	payload := createResp.VirtualHostCreate

	switch payload.Typename {
	case "VirtualHostCreated":
		if payload.VirtualHost == nil {
			return diag.Errorf("virtualHostCreate: backend returned VirtualHostCreated without virtualHost")
		}

		vm := payload.VirtualHost

		d.SetId(vm.ID)
		_ = d.Set("uuid", vm.UUID)
		_ = d.Set("hostname", vm.Hostname)
		_ = d.Set("cpu_count", vm.CpuCount)
		_ = d.Set("cores_per_socket", vm.CoresPerSocket)
		_ = d.Set("memory_size_gb", vm.MemorySizeMB/1024)
		_ = d.Set("status", vm.State)
		_ = d.Set("project_id", vm.Project.ID)
		_ = d.Set("customer_id", vm.Customer.ID)
		_ = d.Set("domain_id", vm.Domain.ID)
		_ = d.Set("tier_id", vm.Tier.ID)
		_ = d.Set("template_id", vm.Template.ID)
		_ = d.Set("region", vm.Region)

		return nil

	case "ValidationErrors":
		// Create returns a union payload; on success we use the returned virtualHost directly (no follow-up listing).
		msg := payload.Message
		if len(payload.Errors) > 0 {
			var details string
			for _, e := range payload.Errors {
				details += fmt.Sprintf("%s: %v; ", e.Field, e.Messages)
			}
			msg = fmt.Sprintf("%s (%s)", msg, details)
		}
		if msg == "" {
			msg = "validation failed without message"
		}
		return diag.Errorf("virtualHostCreate: %s", msg)

	case "Unauthorized", "OperationUnavailable":
		msg := payload.Message
		if msg == "" {
			msg = fmt.Sprintf("%s", payload.Typename)
		}
		return diag.Errorf("virtualHostCreate: %s", msg)

	default:
		return diag.Errorf("virtualHostCreate: unexpected payload type %q", payload.Typename)
	}
}

const queryGetVM = `
query GetVm($id: GlobalID!) {
  virtualHost(id: $id) {
    id
    uuid
    hostname
    state
    cpuCount
    coresPerSocket
    memorySizeMB
    note
    dataProtectionPolicy { id note }
    networkInterfaceList {
      network { id }
      ipv4Addresses { ip prefixlen }
      ipv6Addresses { ip prefixlen }
      startConnected
    }
    tier { id }
    domain { id }
    template { id }
    project { id }
    customer { id }
    region
  }
}
`

// ResourceVirtualHostRead refreshes Terraform state from the API.
func ResourceVirtualHostRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	vars := map[string]interface{}{"id": d.Id()}

	var resp struct {
		VirtualHost *struct {
			ID                   string `json:"id"`
			UUID                 string `json:"uuid"`
			Hostname             string `json:"hostname"`
			State                string `json:"state"`
			CpuCount             int    `json:"cpuCount"`
			CoresPerSocket       int    `json:"coresPerSocket"`
			MemorySizeMB         int    `json:"memorySizeMB"`
			Note                 string `json:"note"`
			DataProtectionPolicy *struct {
				ID   string `json:"id"`
				Note string `json:"note"`
			} `json:"dataProtectionPolicy"`
			NetworkInterfaceList []struct {
				Network       struct{ ID string } `json:"network"`
				IPv4Addresses []struct {
					IP        string `json:"ip"`
					Prefixlen int    `json:"prefixlen"`
				} `json:"ipv4Addresses"`
				IPv6Addresses []struct {
					IP        string `json:"ip"`
					Prefixlen int    `json:"prefixlen"`
				} `json:"ipv6Addresses"`
				StartConnected bool `json:"startConnected"`
			} `json:"networkInterfaceList"`
			Tier     struct{ ID string } `json:"tier"`
			Domain   struct{ ID string } `json:"domain"`
			Template struct{ ID string } `json:"template"`
			Project  struct{ ID string } `json:"project"`
			Customer struct{ ID string } `json:"customer"`
			Region   string              `json:"region"`
		} `json:"virtualHost"`
	}

	if err := client.Do(queryGetVM, vars, &resp); err != nil {
		return diag.FromErr(err)
	}

	if resp.VirtualHost == nil {
		d.SetId("")
		return nil
	}

	vh := resp.VirtualHost

	// Map API network interfaces into Terraform "interfaces" blocks.
	//
	// Provider schema:
	// - network_id (required)
	// - auto_assign_ip (optional, default true)
	// - ip (optional)
	//
	// API exposes networkInterfaceList with a list of ipv4Addresses. For import / state refresh we
	// use a pragmatic mapping:
	_ = d.Set("uuid", vh.UUID)
	_ = d.Set("hostname", vh.Hostname)
	_ = d.Set("status", vh.State)
	_ = d.Set("cpu_count", vh.CpuCount)
	_ = d.Set("cores_per_socket", vh.CoresPerSocket)
	_ = d.Set("memory_size_gb", vh.MemorySizeMB/1024)

	// This is a Terraform-only safety switch controlling whether resize operations are allowed to restart the VM.
	// The backend doesn't store it on the VirtualHost object, so during Read() we keep whatever value Terraform
	// already has (config/state), falling back to the schema default when not set (e.g., during import).
	_ = d.Set("allow_resize_restart", d.Get("allow_resize_restart").(bool))
	_ = d.Set("note", vh.Note)
	if vh.DataProtectionPolicy != nil {
		_ = d.Set("data_protection_policy", vh.DataProtectionPolicy.ID)
	}
	_ = d.Set("tier_id", vh.Tier.ID)
	_ = d.Set("domain_id", vh.Domain.ID)
	_ = d.Set("template_id", vh.Template.ID)
	_ = d.Set("project_id", vh.Project.ID)
	_ = d.Set("customer_id", vh.Customer.ID)
	_ = d.Set("region", vh.Region)

	// Map backend NetworkInterface objects to Terraform "interfaces" blocks.
	//
	// Important: the platform doesn't provide enough information to reliably distinguish
	// "static IP" vs "DHCP" for every VM in all states. Also, we currently don't implement
	// in-place updates for interfaces.
	//
	// To make imports and `terraform plan -generate-config-out=...` work, we only populate
	// `interfaces` from the API when Terraform doesn't already have interfaces in state/config
	// (typical during import). If the user already manages `interfaces` in their HCL, we leave
	// it untouched to avoid spurious diffs.
	if current, ok := d.Get("interfaces").([]interface{}); ok && len(current) == 0 {
		ifaces := make([]interface{}, 0, len(vh.NetworkInterfaceList))
		for _, ni := range vh.NetworkInterfaceList {
			m := map[string]interface{}{
				"network_id": ni.Network.ID,
			}

			// Heuristic mapping:
			// - if backend returns at least one IPv4 address, treat it as a manually assigned IP
			//   (auto_assign_ip=false) and persist the first address into "ip".
			// - otherwise, keep auto_assign_ip=true and omit "ip".
			if len(ni.IPv4Addresses) > 0 {
				m["auto_assign_ip"] = false
				m["ip"] = ni.IPv4Addresses[0].IP
			} else {
				m["auto_assign_ip"] = true
			}
			ifaces = append(ifaces, m)
		}
		_ = d.Set("interfaces", ifaces)
	}

	return nil
}

// Create returns a union payload; on success we use the returned virtualHost directly (no follow-up listing).
type gqlFieldError struct {
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

const mutationResizeVm = `
mutation ResizeVm($input: VirtualHostResizeInput!) {
  virtualHostResize(input: $input) {
    __typename
    ... on TaskExecutionNode {
      id
    }
    ... on ValidationErrors {
      message
      errors {
        field
        messages
      }
    }
    ... on Unauthorized {
      message
    }
    ... on OperationUnavailable {
      message
      reasons
    }
  }
}
`

const mutationUpdateVmTier = `
mutation UpdateVmTier($input: VirtualHostUpdateTierInput!) {
  virtualHostUpdateTier(input: $input) {
    __typename
    ... on TaskExecutionNode {
      id
    }
    ... on ValidationErrors {
      message
      errors {
        field
        messages
      }
    }
    ... on Unauthorized {
      message
    }
    ... on OperationUnavailable {
      message
      reasons
    }
  }
}
`

// ResourceVirtualHostUpdate updates either sizing or tier; applying both in one run is rejected.
func ResourceVirtualHostUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	// Create returns a union payload; on success we use the returned virtualHost directly (no follow-up listing).
	sizingChanged := d.HasChange("cpu_count") || d.HasChange("cores_per_socket") || d.HasChange("memory_size_gb")
	tierChanged := d.HasChange("tier_id")

	changeGroups := 0
	if sizingChanged {
		changeGroups++
	}
	if tierChanged {
		changeGroups++
	}

	// Create returns a union payload; on success we use the returned virtualHost directly (no follow-up listing).
	if changeGroups == 0 {
		return ResourceVirtualHostRead(ctx, d, meta)
	}

	// Simultaneous sizing and tier changes are not supported in a single apply.
	if changeGroups > 1 {
		return diag.Errorf(
			"ocp_virtual_host: simultaneous change of sizing (cpu_count/cores_per_socket/memory_size_gb) and tier_id in a single apply is not supported. " +
				"Please apply sizing changes first, wait for the job to finish, and then apply tier change in a separate terraform apply.",
		)
	}

	// 1) resize – cpu_count / cores_per_socket / memory_size_gb
	if sizingChanged {
		input := map[string]interface{}{
			"virtualHost":  d.Id(),
			"allowRestart": d.Get("allow_resize_restart").(bool),
		}

		if d.HasChange("cpu_count") {
			input["cpuCount"] = d.Get("cpu_count").(int)
		}
		if d.HasChange("cores_per_socket") {
			input["coresPerSocket"] = d.Get("cores_per_socket").(int)
		}
		if d.HasChange("memory_size_gb") {
			input["memorySizeGB"] = d.Get("memory_size_gb").(int)
		}

		var respResize struct {
			VirtualHostResize struct {
				Typename string          `json:"__typename"`
				Message  string          `json:"message,omitempty"`
				Errors   []gqlFieldError `json:"errors,omitempty"`
				Reasons  []string        `json:"reasons,omitempty"`
			} `json:"virtualHostResize"`
		}

		if err := client.Do(mutationResizeVm, map[string]interface{}{
			"input": input,
		}, &respResize); err != nil {
			return diag.FromErr(err)
		}

		p := respResize.VirtualHostResize

		switch p.Typename {
		case "TaskExecutionNode":
			// OK: job started.

		case "ValidationErrors":
			msg := p.Message
			for _, e := range p.Errors {
				msg += fmt.Sprintf(" %s: %v;", e.Field, e.Messages)
			}
			if msg == "" {
				msg = "validation failed without message"
			}
			return diag.Errorf("virtualHostResize: %s", msg)

		case "Unauthorized", "OperationUnavailable":
			msg := p.Message
			if len(p.Reasons) > 0 {
				msg = fmt.Sprintf("%s (reasons=%v)", msg, p.Reasons)
			}
			if msg == "" {
				msg = p.Typename
			}
			return diag.Errorf("virtualHostResize: %s", msg)

		default:
			return diag.Errorf("virtualHostResize: unexpected payload type %q", p.Typename)
		}
	}

	// Tier change (tier_id).
	if tierChanged {
		input := map[string]interface{}{
			"virtualHost": d.Id(),
			"tier":        d.Get("tier_id").(string),
			// isExtended / targetIops are currently left at API defaults.
		}

		var respTier struct {
			VirtualHostUpdateTier struct {
				Typename string          `json:"__typename"`
				Message  string          `json:"message,omitempty"`
				Errors   []gqlFieldError `json:"errors,omitempty"`
				Reasons  []string        `json:"reasons,omitempty"`
			} `json:"virtualHostUpdateTier"`
		}

		if err := client.Do(mutationUpdateVmTier, map[string]interface{}{
			"input": input,
		}, &respTier); err != nil {
			return diag.FromErr(err)
		}

		p := respTier.VirtualHostUpdateTier

		switch p.Typename {
		case "TaskExecutionNode":
			// OK: job started.

		case "ValidationErrors":
			msg := p.Message
			for _, e := range p.Errors {
				msg += fmt.Sprintf(" %s: %v;", e.Field, e.Messages)
			}
			if msg == "" {
				msg = "validation failed without message"
			}
			return diag.Errorf("virtualHostUpdateTier: %s", msg)

		case "Unauthorized", "OperationUnavailable":
			msg := p.Message
			if len(p.Reasons) > 0 {
				msg = fmt.Sprintf("%s (reasons=%v)", msg, p.Reasons)
			}
			if msg == "" {
				msg = p.Typename
			}
			return diag.Errorf("virtualHostUpdateTier: %s", msg)

		default:
			return diag.Errorf("virtualHostUpdateTier: unexpected payload type %q", p.Typename)
		}
	}

	// isExtended / targetIops are currently left at API defaults.
	return ResourceVirtualHostRead(ctx, d, meta)
}

// ResourceVirtualHostDelete deletes the virtual host and clears Terraform state.
func ResourceVirtualHostDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	const mutationDelete = `
mutation DeleteVm($input: VirtualHostDeleteInput!) {
  virtualHostDelete(input: $input) {
    __typename
  }
}
`

	vars := map[string]interface{}{
		"input": map[string]interface{}{
			"virtualHost": d.Id(),
			// isExtended / targetIops are currently left at API defaults.
		},
	}

	if err := client.Do(mutationDelete, vars, &struct {
		VirtualHostDelete struct {
			Typename string `json:"__typename"`
		} `json:"virtualHostDelete"`
	}{}); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

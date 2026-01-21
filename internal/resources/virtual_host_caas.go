package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ocpclient "github.com/davidhrbac/terraform-provider-ocp/internal/client"
)

// ResourceVirtualHostCaas manages "shadow" VM objects for inventory/accounting.
// OCP does not provision nor control the actual VM; the record is linked to an existing
// VM in a vCenter via UUID.
func ResourceVirtualHostCaas() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVirtualHostCaasCreate,
		ReadContext:   resourceVirtualHostCaasRead,
		UpdateContext: resourceVirtualHostCaasUpdate,
		DeleteContext: resourceVirtualHostCaasDelete,

		// Import expects the VirtualHost GlobalID (same value as resource ID).
		// Example:
		//   terraform import ocp_virtual_host_caas.shadow "VmlydHVhbEhvc3ROb2RlOjEyMzQ1"
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Description: "Region.",
				Required:    true,
				ForceNew:    true, // identity of the shadow object
			},

			"vcenter_id": {
				Type:        schema.TypeString,
				Description: "ID of the vCenter that owns the VM.",
				Required:    true,
				ForceNew:    true,
			},

			"project_id": {
				Type:        schema.TypeString,
				Description: "ID of the project.",
				Required:    true,
				ForceNew:    true,
			},

			"tier_id": {
				Type:        schema.TypeString,
				Description: "ID of the storage tier.",
				Required:    true,
			},

			"hostname": {
				Type:        schema.TypeString,
				Description: "Hostname.",
				Required:    true,
				ForceNew:    true,
			},

			"uuid": {
				Type:        schema.TypeString,
				Description: "VM UUID in vCenter.",
				Required:    true,
				ForceNew:    true,
			},

			"note": {
				Type:        schema.TypeString,
				Description: "Note.",
				Required:    true,
			},

			// Convenience computed fields (inventory/UX).
			"customer_id": {
				Type:        schema.TypeString,
				Description: "ID of the customer.",
				Computed:    true,
			},
			"status": {
				Type:        schema.TypeString,
				Description: "Current status.",
				Computed:    true,
			},
		},
	}
}

type caasFieldError struct {
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

// Common shape for CAAS union payloads. Success branch is VirtualHostNode.
type caasPayload struct {
	Typename string           `json:"__typename"`
	Message  string           `json:"message,omitempty"`
	Errors   []caasFieldError `json:"errors,omitempty"`
	Reasons  []string         `json:"reasons,omitempty"`

	// VirtualHostNode fields (only populated on success branch)
	ID       string `json:"id,omitempty"`
	UUID     string `json:"uuid,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Note     string `json:"note,omitempty"`
	State    string `json:"state,omitempty"`
	Region   string `json:"region,omitempty"`

	Tier struct {
		ID string `json:"id"`
	} `json:"tier,omitempty"`

	Project struct {
		ID string `json:"id"`
	} `json:"project,omitempty"`

	Customer struct {
		ID string `json:"id"`
	} `json:"customer,omitempty"`

	Vcenter struct {
		ID   string `json:"id"`
		Name string `json:"name,omitempty"`
	} `json:"vcenter,omitempty"`
}

func formatValidationMessage(p caasPayload) string {
	msg := p.Message
	for _, e := range p.Errors {
		msg += fmt.Sprintf(" %s: %v;", e.Field, e.Messages)
	}
	if msg == "" {
		msg = "validation failed without message"
	}
	return msg
}

const mutationVirtualHostCreateCaas = `
mutation CreateVirtualHostCaas($input: VirtualHostCreateCaasInput!) {
  virtualHostCreateCaas(input: $input) {
    __typename
    ... on VirtualHostNode {
      id
      uuid
      hostname
      note
      state
      region
      tier { id }
      project { id }
      customer { id }
      vcenter { id name }
    }
    ... on ValidationErrors {
      message
      errors { field messages }
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

const mutationVirtualHostUpdateCaas = `
mutation UpdateVirtualHostCaas($input: VirtualHostUpdateCaasInput!) {
  virtualHostUpdateCaas(input: $input) {
    __typename
    ... on VirtualHostNode {
      id
      uuid
      hostname
      note
      state
      region
      tier { id }
      project { id }
      customer { id }
      vcenter { id name }
    }
    ... on ValidationErrors {
      message
      errors { field messages }
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

const mutationVirtualHostDeleteCaas = `
mutation DeleteVirtualHostCaas($input: VirtualHostDeleteCaasInput!) {
  virtualHostDeleteCaas(input: $input) {
    __typename
    ... on VirtualHostNode { id }
    ... on ValidationErrors {
      message
      errors { field messages }
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

const queryGetVirtualHostCaas = `
query GetVirtualHostCaas($id: GlobalID!) {
  virtualHost(id: $id) {
    id
    uuid
    hostname
    note
    state
    region
    tier { id }
    project { id }
    customer { id }
    vcenter { id name }
  }
}
`

func resourceVirtualHostCaasCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	input := map[string]interface{}{
		"vcenter":  d.Get("vcenter_id").(string),
		"hostname": d.Get("hostname").(string),
		"uuid":     d.Get("uuid").(string),
		"note":     d.Get("note").(string),
		"tier":     d.Get("tier_id").(string),
		"project":  d.Get("project_id").(string),
		"region":   d.Get("region").(string),
	}

	var resp struct {
		VirtualHostCreateCaas caasPayload `json:"virtualHostCreateCaas"`
	}

	if err := client.Do(mutationVirtualHostCreateCaas, map[string]interface{}{"input": input}, &resp); err != nil {
		return diag.FromErr(err)
	}

	p := resp.VirtualHostCreateCaas

	switch p.Typename {
	case "VirtualHostNode":
		if p.ID == "" {
			return diag.Errorf("virtualHostCreateCaas: backend returned VirtualHostNode without id")
		}
		d.SetId(p.ID)

		_ = d.Set("uuid", p.UUID)
		_ = d.Set("hostname", p.Hostname)
		_ = d.Set("note", p.Note)
		_ = d.Set("status", p.State)
		_ = d.Set("region", p.Region)
		_ = d.Set("tier_id", p.Tier.ID)
		_ = d.Set("project_id", p.Project.ID)
		_ = d.Set("customer_id", p.Customer.ID)
		_ = d.Set("vcenter_id", p.Vcenter.ID)

		return nil

	case "ValidationErrors":
		return diag.Errorf("virtualHostCreateCaas: %s", formatValidationMessage(p))

	case "Unauthorized", "OperationUnavailable":
		msg := p.Message
		if len(p.Reasons) > 0 {
			msg = fmt.Sprintf("%s (reasons=%v)", msg, p.Reasons)
		}
		if msg == "" {
			msg = p.Typename
		}
		return diag.Errorf("virtualHostCreateCaas: %s", msg)

	default:
		return diag.Errorf("virtualHostCreateCaas: unexpected payload type %q", p.Typename)
	}
}

func resourceVirtualHostCaasRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	var resp struct {
		VirtualHost *struct {
			ID       string              `json:"id"`
			UUID     string              `json:"uuid"`
			Hostname string              `json:"hostname"`
			Note     string              `json:"note"`
			State    string              `json:"state"`
			Region   string              `json:"region"`
			Tier     struct{ ID string } `json:"tier"`
			Project  struct{ ID string } `json:"project"`
			Customer struct{ ID string } `json:"customer"`
			Vcenter  struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"vcenter"`
		} `json:"virtualHost"`
	}

	if err := client.Do(queryGetVirtualHostCaas, map[string]interface{}{"id": d.Id()}, &resp); err != nil {
		return diag.FromErr(err)
	}

	if resp.VirtualHost == nil {
		d.SetId("")
		return nil
	}

	vh := resp.VirtualHost

	_ = d.Set("uuid", vh.UUID)
	_ = d.Set("hostname", vh.Hostname)
	_ = d.Set("note", vh.Note)
	_ = d.Set("status", vh.State)
	_ = d.Set("region", vh.Region)
	_ = d.Set("tier_id", vh.Tier.ID)
	_ = d.Set("project_id", vh.Project.ID)
	_ = d.Set("customer_id", vh.Customer.ID)
	_ = d.Set("vcenter_id", vh.Vcenter.ID)

	return nil
}

func resourceVirtualHostCaasUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	changed := d.HasChange("note") || d.HasChange("tier_id")
	if !changed {
		return resourceVirtualHostCaasRead(ctx, d, meta)
	}

	input := map[string]interface{}{
		"virtualHost": d.Id(),
	}

	if d.HasChange("note") {
		input["note"] = d.Get("note").(string)
	}

	if d.HasChange("tier_id") {
		input["tier"] = d.Get("tier_id").(string)
	}

	var resp struct {
		VirtualHostUpdateCaas caasPayload `json:"virtualHostUpdateCaas"`
	}

	if err := client.Do(mutationVirtualHostUpdateCaas, map[string]interface{}{"input": input}, &resp); err != nil {
		return diag.FromErr(err)
	}

	p := resp.VirtualHostUpdateCaas

	switch p.Typename {
	case "VirtualHostNode":
		// Refresh state
		return resourceVirtualHostCaasRead(ctx, d, meta)

	case "ValidationErrors":
		return diag.Errorf("virtualHostUpdateCaas: %s", formatValidationMessage(p))

	case "Unauthorized", "OperationUnavailable":
		msg := p.Message
		if len(p.Reasons) > 0 {
			msg = fmt.Sprintf("%s (reasons=%v)", msg, p.Reasons)
		}
		if msg == "" {
			msg = p.Typename
		}
		return diag.Errorf("virtualHostUpdateCaas: %s", msg)

	default:
		return diag.Errorf("virtualHostUpdateCaas: unexpected payload type %q", p.Typename)
	}
}

func resourceVirtualHostCaasDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	input := map[string]interface{}{
		"virtualHost": d.Id(),
	}

	var resp struct {
		VirtualHostDeleteCaas caasPayload `json:"virtualHostDeleteCaas"`
	}

	if err := client.Do(mutationVirtualHostDeleteCaas, map[string]interface{}{"input": input}, &resp); err != nil {
		return diag.FromErr(err)
	}

	p := resp.VirtualHostDeleteCaas

	switch p.Typename {
	case "VirtualHostNode":
		d.SetId("")
		return nil

	case "ValidationErrors":
		return diag.Errorf("virtualHostDeleteCaas: %s", formatValidationMessage(p))

	case "Unauthorized", "OperationUnavailable":
		msg := p.Message
		if len(p.Reasons) > 0 {
			msg = fmt.Sprintf("%s (reasons=%v)", msg, p.Reasons)
		}
		if msg == "" {
			msg = p.Typename
		}
		return diag.Errorf("virtualHostDeleteCaas: %s", msg)

	default:
		return diag.Errorf("virtualHostDeleteCaas: unexpected payload type %q", p.Typename)
	}
}

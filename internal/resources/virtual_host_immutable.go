package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ocpclient "github.com/davidhrbac/terraform-provider-ocp/internal/client"
)

// ResourceVirtualHostImmutable manages virtual hosts created with ignition config data.
func ResourceVirtualHostImmutable() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceVirtualHostImmutableCreate,
		ReadContext:   ResourceVirtualHostImmutableRead,
		UpdateContext: ResourceVirtualHostImmutableUpdate,
		DeleteContext: ResourceVirtualHostDelete,

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
			"template_id": {
				Type:        schema.TypeString,
				Description: "ID of the template used to create the virtual host.",
				Required:    true,
				ForceNew:    true,
			},
			"tier_id": {
				Type:        schema.TypeString,
				Description: "ID of the storage tier assigned to the virtual host.",
				Required:    true,
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
			"note": {
				Type:        schema.TypeString,
				Description: "Note.",
				Required:    true,
			},
			"ignition_config_data": {
				Type:        schema.TypeString,
				Description: "Ignition config data.",
				Required:    true,
				Sensitive:   true,
			},
			"ignition_config_data_encoding": {
				Type:        schema.TypeString,
				Description: "Ignition config data encoding.",
				Optional:    true,
				Default:     "BASE64",
			},
			"os_disk_size_gb": {
				Type:        schema.TypeInt,
				Description: "OS disk size gb.",
				Optional:    true,
				Default:     20,
			},
			"data_protection_policy": {
				Type:        schema.TypeString,
				Description: "Data protection policy.",
				Optional:    true,
			},
			"allow_resize_restart": {
				Type:        schema.TypeBool,
				Description: "Allow resize restart.",
				Optional:    true,
				Default:     true,
			},
			"notify_user": {
				Type:        schema.TypeBool,
				Description: "Notify user when deployment ends.",
				Optional:    true,
				Default:     false,
			},
			"cluster_type": {
				Type:        schema.TypeString,
				Description: "Cluster type.",
				Optional:    true,
				Default:     "PRIMARY",
			},
			"version": {
				Type:        schema.TypeString,
				Description: "Deployment version.",
				Optional:    true,
			},
			"anti_affinity": {
				Type:        schema.TypeString,
				Description: "Anti-affinity group.",
				Optional:    true,
			},
			"business_service": {
				Type:        schema.TypeString,
				Description: "Business service.",
				Optional:    true,
			},
			"dedicated_cluster": {
				Type:        schema.TypeString,
				Description: "Dedicated cluster.",
				Optional:    true,
			},
			"dedicated_dr_cluster": {
				Type:        schema.TypeString,
				Description: "Dedicated DR cluster.",
				Optional:    true,
			},
			"interfaces": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_id": {
							Type:        schema.TypeString,
							Description: "ID of the network for this interface.",
							Required:    true,
						},
						"ip_list": {
							Type:        schema.TypeList,
							Description: "IP addresses for this interface.",
							Optional:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"local_disk_list": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"size_gb": {
							Type:        schema.TypeInt,
							Description: "Local disk size gb.",
							Required:    true,
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

const mutationCreateImmutableVM = `
mutation CreateVmImmutable($input: VirtualHostCreateImmutableInput!) {
  virtualHostCreateImmutable(input: $input) {
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

type immutableVirtualHost struct {
	ID             string              `json:"id"`
	UUID           string              `json:"uuid"`
	Hostname       string              `json:"hostname"`
	State          string              `json:"state"`
	CpuCount       int                 `json:"cpuCount"`
	CoresPerSocket int                 `json:"coresPerSocket"`
	MemorySizeMB   int                 `json:"memorySizeMB"`
	Tier           struct{ ID string } `json:"tier"`
	Template       struct{ ID string } `json:"template"`
	Project        struct{ ID string } `json:"project"`
	Customer       struct{ ID string } `json:"customer"`
	Region         string              `json:"region"`
}

type immutableFieldMessages struct {
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

// ResourceVirtualHostImmutableCreate creates a new virtual host via the API.
func ResourceVirtualHostImmutableCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	input := map[string]interface{}{
		"region":                     d.Get("region").(string),
		"customer":                   d.Get("customer_id").(string),
		"project":                    d.Get("project_id").(string),
		"hostname":                   d.Get("hostname").(string),
		"template":                   d.Get("template_id").(string),
		"cpuCount":                   d.Get("cpu_count").(int),
		"coresPerSocket":             d.Get("cores_per_socket").(int),
		"memorySizeGB":               d.Get("memory_size_gb").(int),
		"tier":                       d.Get("tier_id").(string),
		"note":                       d.Get("note").(string),
		"ignitionConfigData":         d.Get("ignition_config_data").(string),
		"ignitionConfigDataEncoding": d.Get("ignition_config_data_encoding").(string),
		"osDiskSizeGB":               d.Get("os_disk_size_gb").(int),
		"notifyUser":                 d.Get("notify_user").(bool),
		"clusterType":                d.Get("cluster_type").(string),
	}

	if v, ok := d.GetOk("data_protection_policy"); ok {
		input["dataProtectionPolicy"] = v.(string)
	}
	if v, ok := d.GetOk("anti_affinity"); ok {
		input["antiAffinity"] = v.(string)
	}
	if v, ok := d.GetOk("business_service"); ok {
		input["businessService"] = v.(string)
	}
	if v, ok := d.GetOk("dedicated_cluster"); ok {
		input["dedicatedCluster"] = v.(string)
	}
	if v, ok := d.GetOk("dedicated_dr_cluster"); ok {
		input["dedicatedDrCluster"] = v.(string)
	}
	if v, ok := d.GetOk("version"); ok {
		input["version"] = v.(string)
	}

	if rawIfaces, ok := d.GetOk("interfaces"); ok {
		ifaceList := rawIfaces.([]interface{})
		if len(ifaceList) > 0 {
			ifaces := make([]map[string]interface{}, 0, len(ifaceList))
			for _, raw := range ifaceList {
				m := raw.(map[string]interface{})
				iface := map[string]interface{}{
					"network": m["network_id"].(string),
				}
				if v, ok := m["ip_list"]; ok {
					ips := v.([]interface{})
					if len(ips) > 0 {
						ipList := make([]string, 0, len(ips))
						for _, ip := range ips {
							ipList = append(ipList, ip.(string))
						}
						iface["ipList"] = ipList
					}
				}
				ifaces = append(ifaces, iface)
			}
			input["interfaceList"] = ifaces
		}
	}

	if rawDisks, ok := d.GetOk("local_disk_list"); ok {
		diskList := rawDisks.([]interface{})
		if len(diskList) > 0 {
			disks := make([]map[string]interface{}, 0, len(diskList))
			for _, raw := range diskList {
				m := raw.(map[string]interface{})
				disks = append(disks, map[string]interface{}{
					"sizeGB": m["size_gb"].(int),
				})
			}
			input["localDiskList"] = disks
		}
	}

	vars := map[string]interface{}{
		"input": input,
	}

	var createResp struct {
		VirtualHostCreateImmutable struct {
			Typename    string                   `json:"__typename"`
			VirtualHost *immutableVirtualHost    `json:"virtualHost,omitempty"`
			Message     string                   `json:"message,omitempty"`
			Errors      []immutableFieldMessages `json:"errors,omitempty"`
			Reasons     []string                 `json:"reasons,omitempty"`
		} `json:"virtualHostCreateImmutable"`
	}

	if err := client.Do(mutationCreateImmutableVM, vars, &createResp); err != nil {
		return diag.FromErr(err)
	}

	payload := createResp.VirtualHostCreateImmutable

	switch payload.Typename {
	case "VirtualHostCreated":
		if payload.VirtualHost == nil {
			return diag.Errorf("virtualHostCreateImmutable: backend returned VirtualHostCreated without virtualHost")
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
		_ = d.Set("tier_id", vm.Tier.ID)
		_ = d.Set("template_id", vm.Template.ID)
		_ = d.Set("region", vm.Region)
		_ = d.Set("note", d.Get("note").(string))
		if v, ok := d.GetOk("data_protection_policy"); ok {
			_ = d.Set("data_protection_policy", v.(string))
		}
		_ = d.Set("ignition_config_data", d.Get("ignition_config_data").(string))
		_ = d.Set("ignition_config_data_encoding", d.Get("ignition_config_data_encoding").(string))
		_ = d.Set("os_disk_size_gb", d.Get("os_disk_size_gb").(int))
		_ = d.Set("notify_user", d.Get("notify_user").(bool))
		_ = d.Set("cluster_type", d.Get("cluster_type").(string))
		if v, ok := d.GetOk("version"); ok {
			_ = d.Set("version", v.(string))
		}
		if v, ok := d.GetOk("anti_affinity"); ok {
			_ = d.Set("anti_affinity", v.(string))
		}
		if v, ok := d.GetOk("business_service"); ok {
			_ = d.Set("business_service", v.(string))
		}
		if v, ok := d.GetOk("dedicated_cluster"); ok {
			_ = d.Set("dedicated_cluster", v.(string))
		}
		if v, ok := d.GetOk("dedicated_dr_cluster"); ok {
			_ = d.Set("dedicated_dr_cluster", v.(string))
		}
		if v, ok := d.GetOk("interfaces"); ok {
			_ = d.Set("interfaces", v)
		}
		if v, ok := d.GetOk("local_disk_list"); ok {
			_ = d.Set("local_disk_list", v)
		}

		return nil

	case "ValidationErrors":
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
		return diag.Errorf("virtualHostCreateImmutable: %s", msg)

	case "Unauthorized", "OperationUnavailable":
		msg := payload.Message
		if msg == "" {
			msg = fmt.Sprintf("%s", payload.Typename)
		}
		return diag.Errorf("virtualHostCreateImmutable: %s", msg)

	default:
		return diag.Errorf("virtualHostCreateImmutable: unexpected payload type %q", payload.Typename)
	}
}

// ResourceVirtualHostImmutableRead refreshes Terraform state from the API.
func ResourceVirtualHostImmutableRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
				ID string `json:"id"`
			} `json:"dataProtectionPolicy"`
			NetworkInterfaceList []struct {
				Network       struct{ ID string } `json:"network"`
				IPv4Addresses []struct {
					IP string `json:"ip"`
				} `json:"ipv4Addresses"`
			} `json:"networkInterfaceList"`
			Tier     struct{ ID string } `json:"tier"`
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

	_ = d.Set("uuid", vh.UUID)
	_ = d.Set("hostname", vh.Hostname)
	_ = d.Set("status", vh.State)
	_ = d.Set("cpu_count", vh.CpuCount)
	_ = d.Set("cores_per_socket", vh.CoresPerSocket)
	_ = d.Set("memory_size_gb", vh.MemorySizeMB/1024)
	_ = d.Set("note", vh.Note)
	if vh.DataProtectionPolicy != nil {
		_ = d.Set("data_protection_policy", vh.DataProtectionPolicy.ID)
	}
	_ = d.Set("tier_id", vh.Tier.ID)
	_ = d.Set("template_id", vh.Template.ID)
	_ = d.Set("project_id", vh.Project.ID)
	_ = d.Set("customer_id", vh.Customer.ID)
	_ = d.Set("region", vh.Region)

	_ = d.Set("allow_resize_restart", d.Get("allow_resize_restart").(bool))
	_ = d.Set("ignition_config_data", d.Get("ignition_config_data").(string))
	_ = d.Set("ignition_config_data_encoding", d.Get("ignition_config_data_encoding").(string))
	_ = d.Set("os_disk_size_gb", d.Get("os_disk_size_gb").(int))
	_ = d.Set("notify_user", d.Get("notify_user").(bool))
	_ = d.Set("cluster_type", d.Get("cluster_type").(string))
	if v, ok := d.GetOk("version"); ok {
		_ = d.Set("version", v.(string))
	}
	if v, ok := d.GetOk("anti_affinity"); ok {
		_ = d.Set("anti_affinity", v.(string))
	}
	if v, ok := d.GetOk("business_service"); ok {
		_ = d.Set("business_service", v.(string))
	}
	if v, ok := d.GetOk("dedicated_cluster"); ok {
		_ = d.Set("dedicated_cluster", v.(string))
	}
	if v, ok := d.GetOk("dedicated_dr_cluster"); ok {
		_ = d.Set("dedicated_dr_cluster", v.(string))
	}
	if v, ok := d.GetOk("local_disk_list"); ok {
		_ = d.Set("local_disk_list", v)
	}

	if current, ok := d.Get("interfaces").([]interface{}); ok && len(current) == 0 {
		ifaces := make([]interface{}, 0, len(vh.NetworkInterfaceList))
		for _, ni := range vh.NetworkInterfaceList {
			m := map[string]interface{}{
				"network_id": ni.Network.ID,
			}
			if len(ni.IPv4Addresses) > 0 {
				ipList := make([]interface{}, 0, len(ni.IPv4Addresses))
				for _, ip := range ni.IPv4Addresses {
					ipList = append(ipList, ip.IP)
				}
				m["ip_list"] = ipList
			}
			ifaces = append(ifaces, m)
		}
		_ = d.Set("interfaces", ifaces)
	}

	return nil
}

// ResourceVirtualHostImmutableUpdate updates either sizing or tier; applying both in one run is rejected.
func ResourceVirtualHostImmutableUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	sizingChanged := d.HasChange("cpu_count") || d.HasChange("cores_per_socket") || d.HasChange("memory_size_gb")
	tierChanged := d.HasChange("tier_id")

	changeGroups := 0
	if sizingChanged {
		changeGroups++
	}
	if tierChanged {
		changeGroups++
	}

	if changeGroups == 0 {
		return ResourceVirtualHostImmutableRead(ctx, d, meta)
	}

	if changeGroups > 1 {
		return diag.Errorf(
			"ocp_virtual_host_immutable: simultaneous change of sizing (cpu_count/cores_per_socket/memory_size_gb) and tier_id in a single apply is not supported. " +
				"Please apply sizing changes first, wait for the job to finish, and then apply tier change in a separate terraform apply.",
		)
	}

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

	if tierChanged {
		input := map[string]interface{}{
			"virtualHost": d.Id(),
			"tier":        d.Get("tier_id").(string),
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

	return ResourceVirtualHostImmutableRead(ctx, d, meta)
}

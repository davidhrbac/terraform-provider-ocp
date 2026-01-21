package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ocpclient "github.com/davidhrbac/terraform-provider-ocp/internal/client"
)

// DataSourceVcenter returns a data source that looks up a vCenter by name within a customer.
func DataSourceVcenter() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceVcenterRead,

		Schema: map[string]*schema.Schema{
			"customer_id": {
				Type:        schema.TypeString,
				Description: "ID of the customer.",
				Required:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the vCenter.",
				Required:    true,
			},

			"id": {
				Type:        schema.TypeString,
				Description: "ID of the vCenter.",
				Computed:    true,
			},
		},
	}
}

const queryVcenterByNameAndCustomer = `
query VcenterByNameAndCustomer($name: StrFilterLookup, $customer: CustomerFilter) {
  vcenterList(filters: { name: $name, customer: $customer, DISTINCT: true }) {
    edges {
      node {
        id
        name
        customer {
          id
          name
        }
      }
    }
  }
}
`

func dataSourceVcenterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	customerID := d.Get("customer_id").(string)
	name := d.Get("name").(string)

	var resp struct {
		VcenterList struct {
			Edges []struct {
				Node struct {
					ID       string `json:"id"`
					Name     string `json:"name"`
					Customer struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"customer"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"vcenterList"`
	}

	vars := map[string]interface{}{
		"name": map[string]interface{}{
			"exact": name,
		},
		"customer": map[string]interface{}{
			"id": map[string]interface{}{
				"exact": customerID,
			},
		},
	}

	if err := client.Do(queryVcenterByNameAndCustomer, vars, &resp); err != nil {
		return diag.FromErr(err)
	}

	edges := resp.VcenterList.Edges

	if len(edges) == 0 {
		return diag.Errorf("no vcenter found with name %q for customer %q", name, customerID)
	}
	if len(edges) > 1 {
		return diag.Errorf("multiple vcenters found with name %q for given customer, please refine", name)
	}

	id := edges[0].Node.ID
	d.SetId(id)
	_ = d.Set("id", id)

	return nil
}

package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ocpclient "github.com/davidhrbac/terraform-provider-ocp/internal/client"
)

// DataSourceCustomer returns a data source that looks up a customer by name.
func DataSourceCustomer() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCustomerRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Description: "Name of the object.",
				Required: true,
			},
			"id": {
				Type:     schema.TypeString,
				Description: "ID of the object.",
				Computed: true,
			},
		},
	}
}

const queryCustomerByName = `
query CustomerByName($name: StrFilterLookup) {
  customerList(filters: { name: $name }) {
    edges {
      node {
        id
        name
      }
    }
  }
}
`

func dataSourceCustomerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	name := d.Get("name").(string)

	vars := map[string]interface{}{
		"name": map[string]interface{}{
			"exact": name,
		},
	}

	var resp struct {
		CustomerList struct {
			Edges []struct {
				Node struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"customerList"`
	}

	if err := client.Do(queryCustomerByName, vars, &resp); err != nil {
		return diag.FromErr(err)
	}

	if len(resp.CustomerList.Edges) == 0 {
		return diag.Errorf("no customer found with name %q", name)
	}
	if len(resp.CustomerList.Edges) > 1 {
		return diag.Errorf("multiple customers found for name %q, please refine", name)
	}

	id := resp.CustomerList.Edges[0].Node.ID
	d.SetId(id)
	d.Set("id", id)

	return nil
}


package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ocpclient "github.com/davidhrbac/terraform-provider-ocp/internal/client"
)

// DataSourceProject returns a data source that looks up a project by name within a customer.
func DataSourceProject() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceProjectRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Description: "Name of the object.",
				Required: true,
			},
			"customer_id": {
				Type:     schema.TypeString,
				Description: "ID of the customer.",
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

const queryProjectByNameAndCustomer = `
query ProjectByNameAndCustomer($name: StrFilterLookup, $customer: CustomerFilter) {
  projectList(filters: { name: $name, customer: $customer }) {
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

func dataSourceProjectRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	name := d.Get("name").(string)
	customerID := d.Get("customer_id").(string)

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

	var resp struct {
		ProjectList struct {
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
		} `json:"projectList"`
	}

	if err := client.Do(queryProjectByNameAndCustomer, vars, &resp); err != nil {
		return diag.FromErr(err)
	}

	edges := resp.ProjectList.Edges

	if len(edges) == 0 {
		return diag.Errorf("no project found with name %q for customer %q", name, customerID)
	}
	if len(edges) > 1 {
		return diag.Errorf("multiple projects found with name %q for given customer, please refine", name)
	}

	id := edges[0].Node.ID
	d.SetId(id)
	_ = d.Set("id", id)

	return nil
}

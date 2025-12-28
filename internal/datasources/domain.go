package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ocpclient "github.com/davidhrbac/terraform-provider-ocp/internal/client"
)

// DataSourceDomain returns a data source that looks up a domain by name.
func DataSourceDomain() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDomainRead,

		Schema: map[string]*schema.Schema{
			"customer_id": {
				Type:        schema.TypeString,
				Description: "ID of the customer.",
				Required:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the object.",
				Required:    true,
			},
			"id": {
				Type:        schema.TypeString,
				Description: "ID of the object.",
				Computed:    true,
			},
		},
	}
}

const queryDomainByFilters = `
query DomainByFilters($filters: DomainFilter) {
  domainList(filters: $filters, first: 100) {
    edges {
      node {
        id
        name
      }
    }
  }
}
`

func dataSourceDomainRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	customerID := d.Get("customer_id").(string)
	name := d.Get("name").(string)

	vars := map[string]interface{}{
		"filters": map[string]interface{}{
			"customer": map[string]interface{}{
				"id": map[string]interface{}{
					"exact": customerID,
				},
			},
			"name": map[string]interface{}{
				"exact": name,
			},
		},
	}

	var resp struct {
		DomainList struct {
			Edges []struct {
				Node struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"domainList"`
	}

	if err := client.Do(queryDomainByFilters, vars, &resp); err != nil {
		return diag.FromErr(err)
	}

	edges := resp.DomainList.Edges

	if len(edges) == 0 {
		return diag.Errorf(
			"no domain found for customer %q with name %q",
			customerID, name,
		)
	}

	if len(edges) > 1 {
		return diag.Errorf(
			"multiple domains found for customer %q with name %q",
			customerID, name,
		)
	}

	node := edges[0].Node

	d.SetId(node.ID)
	_ = d.Set("id", node.ID)

	return nil
}

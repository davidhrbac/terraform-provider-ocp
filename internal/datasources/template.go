package datasources

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ocpclient "github.com/davidhrbac/terraform-provider-ocp/internal/client"
)

// DataSourceTemplate returns a data source that looks up a template by name.
func DataSourceTemplate() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceTemplateRead,

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
			"region": {
				Type:     schema.TypeString,
				Description: "Region.",
				Required: true,
			},
			"solution_type": {
				Type:     schema.TypeString,
				Description: "Solution type.",
				Optional: true,
			},
			"id": {
				Type:     schema.TypeString,
				Description: "ID of the object.",
				Computed: true,
			},
		},
	}
}

const queryTemplateByName = `
query TemplateByName($filters: TemplateFilter) {
  templateList(filters: $filters) {
    edges {
      node {
        id
        name
      }
    }
  }
}
`

func dataSourceTemplateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	name := d.Get("name").(string)
	customerID := d.Get("customer_id").(string)
	region := strings.ToUpper(d.Get("region").(string))

	solutionType := "OCP"
	if v, ok := d.GetOk("solution_type"); ok {
		solutionType = strings.ToUpper(v.(string))
	}

	nameFilter := map[string]interface{}{
		"exact": name,
	}

	customerFilter := map[string]interface{}{
		"id": map[string]interface{}{
			"exact": customerID,
		},
	}

	solutionTypeFilter := map[string]interface{}{
		"exact": solutionType,
	}

	regionFilter := map[string]interface{}{
		"exact": region,
	}

	filters := map[string]interface{}{
		"name":         nameFilter,
		"customer":     customerFilter,
		"solutionType": solutionTypeFilter,
		"region":       regionFilter,
	}

	vars := map[string]interface{}{
		"filters": filters,
	}

	var resp struct {
		TemplateList struct {
			Edges []struct {
				Node struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"templateList"`
	}

	if err := client.Do(queryTemplateByName, vars, &resp); err != nil {
		return diag.FromErr(err)
	}

	edges := resp.TemplateList.Edges

	if len(edges) == 0 {
		return diag.Errorf(
			"no template found with name %q for customer %q in region %q (solution_type %q)",
			name, customerID, region, solutionType,
		)
	}

	if len(edges) > 1 {
		return diag.Errorf(
			"multiple templates found with name %q for customer %q in region %q (solution_type %q), must be unique",
			name, customerID, region, solutionType,
		)
	}

	id := edges[0].Node.ID

	d.SetId(id)
	_ = d.Set("id", id)

	return nil
}

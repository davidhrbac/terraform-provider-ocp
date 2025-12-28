package datasources

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ocpclient "github.com/davidhrbac/terraform-provider-ocp/internal/client"
)

// DataSourceDataProtectionPolicy returns a data source that looks up a data protection policy by name.
func DataSourceDataProtectionPolicy() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataProtectionPolicyRead,

		Schema: map[string]*schema.Schema{
			"customer_id": {
				Type:        schema.TypeString,
				Description: "ID of the customer.",
				Required:    true,
			},
			"project_id": {
				Type:        schema.TypeString,
				Description: "ID of the project.",
				Required:    true,
			},
			"note": {
				Type:        schema.TypeString,
				Description: "Note.",
				Required:    true,
			},
			"solution_type": {
				Type:        schema.TypeString,
				Description: "Solution type.",
				Optional:    true,
			},
			"id": {
				Type:        schema.TypeString,
				Description: "ID of the object.",
				Computed:    true,
			},
		},
	}
}

const queryDataProtectionPolicyByFilters = `
query DataProtectionPolicyByFilters($filters: DataProtectionPolicyFilter) {
  dataProtectionPolicyList(filters: $filters, first: 100) {
    edges {
      node {
        id
        note
        customer {
          id
          name
        }
      }
    }
  }
}
`

func dataProtectionPolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	customerID := d.Get("customer_id").(string)
	projectID := d.Get("project_id").(string)
	note := d.Get("note").(string)

	solutionType := "OCP"
	if v, ok := d.GetOk("solution_type"); ok {
		solutionType = strings.ToUpper(v.(string))
	}

	customerFilter := map[string]interface{}{
		"id": map[string]interface{}{
			"exact": customerID,
		},
		"projectList": map[string]interface{}{
			"id": map[string]interface{}{
				"exact": projectID,
			},
		},
	}

	separationPodFilter := map[string]interface{}{
		"solutionType": map[string]interface{}{
			"exact": solutionType,
		},
	}

	dedicatedClusterFilter := map[string]interface{}{
		"id": map[string]interface{}{
			"isNull": true,
		},
	}

	noteFilter := map[string]interface{}{
		"exact": note,
	}

	filters := map[string]interface{}{
		"customer":          customerFilter,
		"separationPodList": separationPodFilter,
		"dedicatedCluster":  dedicatedClusterFilter,
		"note":              noteFilter,
		"DISTINCT":          true,
	}

	vars := map[string]interface{}{
		"filters": filters,
	}

	var resp struct {
		DataProtectionPolicyList struct {
			Edges []struct {
				Node struct {
					ID       string `json:"id"`
					Note     string `json:"note"`
					Customer struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"customer"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"dataProtectionPolicyList"`
	}

	if err := client.Do(queryDataProtectionPolicyByFilters, vars, &resp); err != nil {
		return diag.FromErr(err)
	}

	edges := resp.DataProtectionPolicyList.Edges

	if len(edges) == 0 {
		return diag.Errorf(
			"no data protection policy found for customer_id=%q, project_id=%q, solution_type=%q, note=%q",
			customerID, projectID, solutionType, note,
		)
	}

	if len(edges) > 1 {
		return diag.Errorf(
			"multiple data protection policies found for customer_id=%q, project_id=%q, solution_type=%q, note=%q (after DISTINCT); must be unique",
			customerID, projectID, solutionType, note,
		)
	}

	node := edges[0].Node

	d.SetId(node.ID)
	_ = d.Set("id", node.ID)

	return nil
}

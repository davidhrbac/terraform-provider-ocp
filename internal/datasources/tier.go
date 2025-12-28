package datasources

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ocpclient "github.com/davidhrbac/terraform-provider-ocp/internal/client"
)

// DataSourceTier returns a data source that looks up a tier by name.
func DataSourceTier() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceTierRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the object.",
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

const queryTierByName = `
query TierByName($name: StrFilterLookup, $solutionType: SolutionTypeEnumFilterLookup) {
  tierList(filters: { name: $name, solutionType: $solutionType }) {
    edges {
      node {
        id
        name
      }
    }
  }
}
`

func dataSourceTierRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ocpclient.Client)

	name := d.Get("name").(string)

	solutionType := "OCP"
	if v, ok := d.GetOk("solution_type"); ok {
		solutionType = strings.ToUpper(v.(string))
	}

	nameFilter := map[string]interface{}{
		"exact": name,
	}

	solutionTypeFilter := map[string]interface{}{
		"exact": solutionType,
	}

	vars := map[string]interface{}{
		"name":         nameFilter,
		"solutionType": solutionTypeFilter,
	}

	var resp struct {
		TierList struct {
			Edges []struct {
				Node struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"tierList"`
	}

	if err := client.Do(queryTierByName, vars, &resp); err != nil {
		return diag.FromErr(err)
	}

	edges := resp.TierList.Edges

	if len(edges) == 0 {
		return diag.Errorf("no tier found with name %q for solution_type %q", name, solutionType)
	}

	if len(edges) > 1 {
		return diag.Errorf(
			"multiple tiers found with name %q for solution_type %q, must be unique",
			name, solutionType,
		)
	}

	id := edges[0].Node.ID

	d.SetId(id)
	_ = d.Set("id", id)

	return nil
}

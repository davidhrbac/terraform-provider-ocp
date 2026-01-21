package datasources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ocpclient "github.com/davidhrbac/terraform-provider-ocp/internal/client"
)

func TestDataSourceVcenterRead(t *testing.T) {
	testCases := []struct {
		name        string
		edges       []map[string]interface{}
		wantError   bool
		errorSubstr string
	}{
		{
			name: "success",
			edges: []map[string]interface{}{
				{
					"node": map[string]interface{}{
						"id":   "vc-1",
						"name": "vcenter-01",
						"customer": map[string]interface{}{
							"id":   "cust-1",
							"name": "customer-a",
						},
					},
				},
			},
		},
		{
			name:        "not found",
			edges:       []map[string]interface{}{},
			wantError:   true,
			errorSubstr: "no vcenter found",
		},
		{
			name: "multiple",
			edges: []map[string]interface{}{
				{"node": map[string]interface{}{"id": "vc-1"}},
				{"node": map[string]interface{}{"id": "vc-2"}},
			},
			wantError:   true,
			errorSubstr: "multiple vcenters found",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := map[string]interface{}{
					"data": map[string]interface{}{
						"vcenterList": map[string]interface{}{
							"edges": tc.edges,
						},
					},
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Fatalf("encode response: %v", err)
				}
			}))
			defer server.Close()

			client := ocpclient.New(server.URL, "token", true)
			data := schema.TestResourceDataRaw(t, DataSourceVcenter().Schema, map[string]interface{}{
				"customer_id": "cust-1",
				"name":        "vcenter-01",
			})

			diags := dataSourceVcenterRead(context.Background(), data, client)
			if tc.wantError {
				if !diags.HasError() {
					t.Fatalf("expected error, got none")
				}
				if tc.errorSubstr != "" && !strings.Contains(diags[0].Summary, tc.errorSubstr) {
					t.Fatalf("expected error containing %q, got %q", tc.errorSubstr, diags[0].Summary)
				}
				return
			}

			if diags.HasError() {
				t.Fatalf("unexpected error: %v", diags[0].Summary)
			}
			if data.Id() != "vc-1" {
				t.Fatalf("expected id vc-1, got %q", data.Id())
			}
			if got := data.Get("id").(string); got != "vc-1" {
				t.Fatalf("expected id attribute vc-1, got %q", got)
			}
		})
	}
}

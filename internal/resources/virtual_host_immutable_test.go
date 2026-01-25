package resources

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

func TestResourceVirtualHostImmutableCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(body.Query, "virtualHostCreateImmutable") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"virtualHostCreateImmutable": map[string]interface{}{
					"__typename": "VirtualHostCreated",
					"virtualHost": map[string]interface{}{
						"id":             "vh-immutable-1",
						"uuid":           "vh-uuid-1",
						"hostname":       "immutable-vm",
						"state":          "ACTIVE",
						"cpuCount":       4,
						"coresPerSocket": 1,
						"memorySizeMB":   16384,
						"tier": map[string]interface{}{
							"id": "tier-1",
						},
						"template": map[string]interface{}{
							"id": "template-1",
						},
						"project": map[string]interface{}{
							"id": "project-1",
						},
						"customer": map[string]interface{}{
							"id": "customer-1",
						},
						"region": "FINLAND",
					},
				},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := ocpclient.New(server.URL, "token", true)
	data := schema.TestResourceDataRaw(t, ResourceVirtualHostImmutable().Schema, map[string]interface{}{
		"region":               "FINLAND",
		"customer_id":          "customer-1",
		"project_id":           "project-1",
		"hostname":             "immutable-vm",
		"template_id":          "template-1",
		"tier_id":              "tier-1",
		"cpu_count":            4,
		"memory_size_gb":       16,
		"note":                 "managed-by-terraform",
		"ignition_config_data": "aWduaXRpb24=",
	})

	diags := ResourceVirtualHostImmutableCreate(context.Background(), data, client)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags[0].Summary)
	}

	if data.Id() != "vh-immutable-1" {
		t.Fatalf("expected id vh-immutable-1, got %q", data.Id())
	}
	if got := data.Get("status").(string); got != "ACTIVE" {
		t.Fatalf("expected status ACTIVE, got %q", got)
	}
}

func TestResourceVirtualHostImmutableReadNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"virtualHost": nil,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := ocpclient.New(server.URL, "token", true)
	data := schema.TestResourceDataRaw(t, ResourceVirtualHostImmutable().Schema, map[string]interface{}{
		"region":               "FINLAND",
		"customer_id":          "customer-1",
		"project_id":           "project-1",
		"hostname":             "immutable-vm",
		"template_id":          "template-1",
		"tier_id":              "tier-1",
		"cpu_count":            4,
		"memory_size_gb":       16,
		"note":                 "managed-by-terraform",
		"ignition_config_data": "aWduaXRpb24=",
	})
	data.SetId("vh-immutable-1")

	diags := ResourceVirtualHostImmutableRead(context.Background(), data, client)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags[0].Summary)
	}
	if data.Id() != "" {
		t.Fatalf("expected id to be cleared, got %q", data.Id())
	}
}

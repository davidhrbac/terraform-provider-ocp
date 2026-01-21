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

func TestResourceVirtualHostCaasCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(body.Query, "virtualHostCreateCaas") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"virtualHostCreateCaas": map[string]interface{}{
					"__typename": "VirtualHostNode",
					"id":         "vh-1",
					"uuid":       "legacy-vm",
					"hostname":   "legacy-vm",
					"note":       "inventory-only",
					"state":      "ACTIVE",
					"region":     "FINLAND",
					"tier": map[string]interface{}{
						"id": "tier-1",
					},
					"project": map[string]interface{}{
						"id": "project-1",
					},
					"customer": map[string]interface{}{
						"id": "customer-1",
					},
					"vcenter": map[string]interface{}{
						"id":   "vcenter-1",
						"name": "vcenter-01",
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
	data := schema.TestResourceDataRaw(t, ResourceVirtualHostCaas().Schema, map[string]interface{}{
		"region":     "FINLAND",
		"vcenter_id": "vcenter-1",
		"project_id": "project-1",
		"tier_id":    "tier-1",
		"hostname":   "legacy-vm",
		"uuid":       "legacy-vm",
		"note":       "inventory-only",
	})

	diags := resourceVirtualHostCaasCreate(context.Background(), data, client)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags[0].Summary)
	}

	if data.Id() != "vh-1" {
		t.Fatalf("expected id vh-1, got %q", data.Id())
	}
	if got := data.Get("customer_id").(string); got != "customer-1" {
		t.Fatalf("expected customer_id customer-1, got %q", got)
	}
	if got := data.Get("status").(string); got != "ACTIVE" {
		t.Fatalf("expected status ACTIVE, got %q", got)
	}
}

func TestResourceVirtualHostCaasReadNotFound(t *testing.T) {
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
	data := schema.TestResourceDataRaw(t, ResourceVirtualHostCaas().Schema, map[string]interface{}{
		"region":     "FINLAND",
		"vcenter_id": "vcenter-1",
		"project_id": "project-1",
		"tier_id":    "tier-1",
		"hostname":   "legacy-vm",
		"uuid":       "legacy-vm",
		"note":       "inventory-only",
	})
	data.SetId("vh-1")

	diags := resourceVirtualHostCaasRead(context.Background(), data, client)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags[0].Summary)
	}
	if data.Id() != "" {
		t.Fatalf("expected id to be cleared, got %q", data.Id())
	}
}

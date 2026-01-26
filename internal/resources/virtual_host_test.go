package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	ocpclient "github.com/davidhrbac/terraform-provider-ocp/internal/client"
)

func resourceDataWithState(t *testing.T, res *schema.Resource, state *terraform.InstanceState, raw map[string]interface{}) *schema.ResourceData {
	t.Helper()

	sm := schema.InternalMap(res.Schema)
	cfg := terraform.NewResourceConfigRaw(raw)
	diff, err := sm.Diff(context.Background(), state, cfg, nil, nil, true)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}

	rd, err := sm.Data(state, diff)
	if err != nil {
		t.Fatalf("data: %v", err)
	}

	return rd
}

func TestResourceVirtualHostCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(body.Query, "virtualHostCreate") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"virtualHostCreate": map[string]interface{}{
					"__typename": "VirtualHostCreated",
					"virtualHost": map[string]interface{}{
						"id":             "vh-1",
						"uuid":           "uuid-1",
						"hostname":       "app-1",
						"state":          "ACTIVE",
						"cpuCount":       2,
						"coresPerSocket": 1,
						"memorySizeMB":   8192,
						"tier": map[string]interface{}{
							"id": "tier-1",
						},
						"domain": map[string]interface{}{
							"id": "domain-1",
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
	data := schema.TestResourceDataRaw(t, ResourceVirtualHost().Schema, map[string]interface{}{
		"region":                 "FINLAND",
		"customer_id":            "customer-1",
		"project_id":             "project-1",
		"hostname":               "app-1",
		"domain_id":              "domain-1",
		"cpu_count":              2,
		"cores_per_socket":       1,
		"memory_size_gb":         8,
		"tier_id":                "tier-1",
		"template_id":            "template-1",
		"note":                   "managed-by-terraform",
		"data_protection_policy": "policy-1",
		"interfaces": []interface{}{
			map[string]interface{}{
				"network_id":     "net-1",
				"auto_assign_ip": true,
			},
		},
	})

	diags := ResourceVirtualHostCreate(context.Background(), data, client)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags[0].Summary)
	}

	if data.Id() != "vh-1" {
		t.Fatalf("expected id vh-1, got %q", data.Id())
	}
	if got := data.Get("status").(string); got != "ACTIVE" {
		t.Fatalf("expected status ACTIVE, got %q", got)
	}
	if got := data.Get("memory_size_gb").(int); got != 8 {
		t.Fatalf("expected memory_size_gb 8, got %d", got)
	}
}

func TestResourceVirtualHostReadNotFound(t *testing.T) {
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
	data := schema.TestResourceDataRaw(t, ResourceVirtualHost().Schema, map[string]interface{}{
		"region":                 "FINLAND",
		"customer_id":            "customer-1",
		"project_id":             "project-1",
		"hostname":               "app-1",
		"domain_id":              "domain-1",
		"cpu_count":              2,
		"cores_per_socket":       1,
		"memory_size_gb":         8,
		"tier_id":                "tier-1",
		"template_id":            "template-1",
		"note":                   "managed-by-terraform",
		"data_protection_policy": "policy-1",
		"interfaces":             []interface{}{},
	})
	data.SetId("vh-1")

	diags := ResourceVirtualHostRead(context.Background(), data, client)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags[0].Summary)
	}
	if data.Id() != "" {
		t.Fatalf("expected id to be cleared, got %q", data.Id())
	}
}

func TestResourceVirtualHostReadMapsInterfaces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"virtualHost": map[string]interface{}{
					"id":             "vh-1",
					"uuid":           "uuid-1",
					"hostname":       "app-1",
					"state":          "ACTIVE",
					"cpuCount":       2,
					"coresPerSocket": 1,
					"memorySizeMB":   8192,
					"note":           "managed-by-terraform",
					"dataProtectionPolicy": map[string]interface{}{
						"id":   "policy-1",
						"note": "daily",
					},
					"networkInterfaceList": []interface{}{
						map[string]interface{}{
							"network": map[string]interface{}{
								"id": "net-1",
							},
							"ipv4Addresses": []interface{}{
								map[string]interface{}{
									"ip":        "192.0.2.10",
									"prefixlen": 24,
								},
							},
							"ipv6Addresses":  []interface{}{},
							"startConnected": true,
						},
					},
					"tier": map[string]interface{}{
						"id": "tier-1",
					},
					"domain": map[string]interface{}{
						"id": "domain-1",
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
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := ocpclient.New(server.URL, "token", true)
	data := schema.TestResourceDataRaw(t, ResourceVirtualHost().Schema, map[string]interface{}{
		"region":                 "FINLAND",
		"customer_id":            "customer-1",
		"project_id":             "project-1",
		"hostname":               "app-1",
		"domain_id":              "domain-1",
		"cpu_count":              2,
		"cores_per_socket":       1,
		"memory_size_gb":         8,
		"tier_id":                "tier-1",
		"template_id":            "template-1",
		"note":                   "managed-by-terraform",
		"data_protection_policy": "policy-1",
		"allow_resize_restart":   false,
		"interfaces":             []interface{}{},
	})
	data.SetId("vh-1")

	diags := ResourceVirtualHostRead(context.Background(), data, client)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags[0].Summary)
	}

	if got := data.Get("allow_resize_restart").(bool); got {
		t.Fatalf("expected allow_resize_restart false, got true")
	}

	if got := data.Get("data_protection_policy").(string); got != "policy-1" {
		t.Fatalf("expected data_protection_policy policy-1, got %q", got)
	}

	ifaces := data.Get("interfaces").([]interface{})
	if len(ifaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(ifaces))
	}
	iface := ifaces[0].(map[string]interface{})
	if got := iface["network_id"].(string); got != "net-1" {
		t.Fatalf("expected network_id net-1, got %q", got)
	}
	if got := iface["auto_assign_ip"].(bool); got {
		t.Fatalf("expected auto_assign_ip false, got true")
	}
	if got := iface["ip"].(string); got != "192.0.2.10" {
		t.Fatalf("expected ip 192.0.2.10, got %q", got)
	}
}

func TestResourceVirtualHostUpdateResize(t *testing.T) {
	var resizeCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch {
		case strings.Contains(body.Query, "virtualHostResize"):
			resizeCalled = true
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"virtualHostResize": map[string]interface{}{
						"__typename": "TaskExecutionNode",
						"id":         "task-1",
					},
				},
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Fatalf("encode response: %v", err)
			}
		case strings.Contains(body.Query, "virtualHost(id"):
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"virtualHost": map[string]interface{}{
						"id":             "vh-1",
						"uuid":           "uuid-1",
						"hostname":       "app-1",
						"state":          "ACTIVE",
						"cpuCount":       4,
						"coresPerSocket": 1,
						"memorySizeMB":   8192,
						"note":           "managed-by-terraform",
						"dataProtectionPolicy": map[string]interface{}{
							"id":   "policy-1",
							"note": "daily",
						},
						"networkInterfaceList": []interface{}{},
						"tier": map[string]interface{}{
							"id": "tier-1",
						},
						"domain": map[string]interface{}{
							"id": "domain-1",
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
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Fatalf("encode response: %v", err)
			}
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	res := ResourceVirtualHost()
	oldData := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{
		"region":                 "FINLAND",
		"customer_id":            "customer-1",
		"project_id":             "project-1",
		"hostname":               "app-1",
		"domain_id":              "domain-1",
		"cpu_count":              2,
		"cores_per_socket":       1,
		"memory_size_gb":         8,
		"tier_id":                "tier-1",
		"template_id":            "template-1",
		"note":                   "managed-by-terraform",
		"data_protection_policy": "policy-1",
		"interfaces": []interface{}{
			map[string]interface{}{"network_id": "net-1"},
		},
	})
	oldData.SetId("vh-1")
	state := oldData.State()

	newData := resourceDataWithState(t, res, state, map[string]interface{}{
		"region":                 "FINLAND",
		"customer_id":            "customer-1",
		"project_id":             "project-1",
		"hostname":               "app-1",
		"domain_id":              "domain-1",
		"cpu_count":              4,
		"cores_per_socket":       1,
		"memory_size_gb":         8,
		"tier_id":                "tier-1",
		"template_id":            "template-1",
		"note":                   "managed-by-terraform",
		"data_protection_policy": "policy-1",
		"interfaces": []interface{}{
			map[string]interface{}{"network_id": "net-1"},
		},
	})
	newData.SetId("vh-1")

	client := ocpclient.New(server.URL, "token", true)
	diags := ResourceVirtualHostUpdate(context.Background(), newData, client)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags[0].Summary)
	}
	if !resizeCalled {
		t.Fatalf("expected resize mutation to be called")
	}
}

func TestResourceVirtualHostUpdateTier(t *testing.T) {
	var tierCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch {
		case strings.Contains(body.Query, "virtualHostUpdateTier"):
			tierCalled = true
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"virtualHostUpdateTier": map[string]interface{}{
						"__typename": "TaskExecutionNode",
						"id":         "task-2",
					},
				},
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Fatalf("encode response: %v", err)
			}
		case strings.Contains(body.Query, "virtualHost(id"):
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"virtualHost": map[string]interface{}{
						"id":             "vh-1",
						"uuid":           "uuid-1",
						"hostname":       "app-1",
						"state":          "ACTIVE",
						"cpuCount":       2,
						"coresPerSocket": 1,
						"memorySizeMB":   8192,
						"note":           "managed-by-terraform",
						"dataProtectionPolicy": map[string]interface{}{
							"id":   "policy-1",
							"note": "daily",
						},
						"networkInterfaceList": []interface{}{},
						"tier": map[string]interface{}{
							"id": "tier-2",
						},
						"domain": map[string]interface{}{
							"id": "domain-1",
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
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Fatalf("encode response: %v", err)
			}
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	res := ResourceVirtualHost()
	oldData := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{
		"region":                 "FINLAND",
		"customer_id":            "customer-1",
		"project_id":             "project-1",
		"hostname":               "app-1",
		"domain_id":              "domain-1",
		"cpu_count":              2,
		"cores_per_socket":       1,
		"memory_size_gb":         8,
		"tier_id":                "tier-1",
		"template_id":            "template-1",
		"note":                   "managed-by-terraform",
		"data_protection_policy": "policy-1",
		"interfaces": []interface{}{
			map[string]interface{}{"network_id": "net-1"},
		},
	})
	oldData.SetId("vh-1")
	state := oldData.State()

	newData := resourceDataWithState(t, res, state, map[string]interface{}{
		"region":                 "FINLAND",
		"customer_id":            "customer-1",
		"project_id":             "project-1",
		"hostname":               "app-1",
		"domain_id":              "domain-1",
		"cpu_count":              2,
		"cores_per_socket":       1,
		"memory_size_gb":         8,
		"tier_id":                "tier-2",
		"template_id":            "template-1",
		"note":                   "managed-by-terraform",
		"data_protection_policy": "policy-1",
		"interfaces": []interface{}{
			map[string]interface{}{"network_id": "net-1"},
		},
	})
	newData.SetId("vh-1")

	client := ocpclient.New(server.URL, "token", true)
	diags := ResourceVirtualHostUpdate(context.Background(), newData, client)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags[0].Summary)
	}
	if !tierCalled {
		t.Fatalf("expected update tier mutation to be called")
	}
}

func TestResourceVirtualHostDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(body.Query, "virtualHostDelete") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"virtualHostDelete": map[string]interface{}{
					"__typename": "TaskExecutionNode",
				},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := ocpclient.New(server.URL, "token", true)
	data := schema.TestResourceDataRaw(t, ResourceVirtualHost().Schema, map[string]interface{}{
		"region":                 "FINLAND",
		"customer_id":            "customer-1",
		"project_id":             "project-1",
		"hostname":               "app-1",
		"domain_id":              "domain-1",
		"cpu_count":              2,
		"cores_per_socket":       1,
		"memory_size_gb":         8,
		"tier_id":                "tier-1",
		"template_id":            "template-1",
		"note":                   "managed-by-terraform",
		"data_protection_policy": "policy-1",
		"interfaces": []interface{}{
			map[string]interface{}{"network_id": "net-1"},
		},
	})
	data.SetId("vh-1")

	diags := ResourceVirtualHostDelete(context.Background(), data, client)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags[0].Summary)
	}
	if data.Id() != "" {
		t.Fatalf("expected id to be cleared, got %q", data.Id())
	}
}

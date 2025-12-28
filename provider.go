// Terraform provider for OCP GraphQL API.
//
// This package wires Terraform SDK resources/data sources to an API client configured
// via provider settings (endpoint, token, TLS verification).
package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ocpclient "github.com/davidhrbac/terraform-provider-ocp/internal/client"
	"github.com/davidhrbac/terraform-provider-ocp/internal/datasources"
	"github.com/davidhrbac/terraform-provider-ocp/internal/resources"
)

type OCPClient struct {
	Endpoint string
	Token    string
	Insecure bool
}

// Provider returns the terraform-plugin-sdk Provider definition, including configuration schema, resources, data sources, and client initialization.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"endpoint": {
				Type:        schema.TypeString,
				Description: "Base URL of the OCP GraphQL API.",
				Optional:    true,
				Default:     "https://ocpportal.int.tieto.com/v2/graphql/",
			},
			"token": {
				Type:        schema.TypeString,
				Description: "Authentication token for the OCP GraphQL API.",
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("OCP_TOKEN", nil),
			},
			"insecure_skip_verify": {
				Type:        schema.TypeBool,
				Description: "Insecure skip verify.",
				Optional:    true,
				Default:     true,
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"ocp_customer":               datasources.DataSourceCustomer(),
			"ocp_data_protection_policy": datasources.DataSourceDataProtectionPolicy(),
			"ocp_domain":                 datasources.DataSourceDomain(),
			"ocp_network":                datasources.DataSourceNetwork(),
			"ocp_project":                datasources.DataSourceProject(),
			"ocp_template":               datasources.DataSourceTemplate(),
			"ocp_tier":                   datasources.DataSourceTier(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"ocp_virtual_host": resources.ResourceVirtualHost(),
		},

		// ConfigureContextFunc initializes the API client once and stores it in `meta`.
		// All resources and data sources retrieve it via `meta.(*client.Client)`.
		//
		// Token is required either via provider configuration or OCP_TOKEN environment variable.
		ConfigureContextFunc: func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
			endpoint := d.Get("endpoint").(string)
			token := d.Get("token").(string)
			insecure := d.Get("insecure_skip_verify").(bool)

			var diags diag.Diagnostics

			// ConfigureContextFunc initializes the API client once and stores it in `meta`.
			// All resources and data sources retrieve it via `meta.(*client.Client)`.
			//
			// Token is required either via provider configuration or OCP_TOKEN environment variable.
			if token == "" {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "Missing OCP token",
					Detail:   "Either set provider `token` or environment variable OCP_TOKEN.",
				})
				return nil, diags
			}

			client := ocpclient.New(endpoint, token, insecure)

			return client, diags
		},
	}
}

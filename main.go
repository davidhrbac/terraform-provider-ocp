package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

// main is the provider entrypoint. Terraform loads the provider via the plugin protocol.
func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: Provider,
	})
}

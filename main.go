package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/rafftechnologies/terraform-provider-raff/internal/provider"
)

// version is stamped at release time by goreleaser
// (-ldflags "-X main.version=..."). Local and `go build` binaries keep "dev".
var version = "dev"

func main() {
	// The provider reports this in its User-Agent, so the API logs show which
	// provider version made a request.
	provider.Version = version

	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: provider.New,
	})
}

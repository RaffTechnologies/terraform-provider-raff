# Raff Terraform Provider

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Terraform provider for [Raff Cloud](https://rafftechnologies.com), built on [raff-go](https://github.com/RaffTechnologies/raff-go).

> **Pre-release:** the provider has not yet been published to the Terraform Registry. To use it today, build locally and add a `dev_overrides` block (see [Building The Provider](#building-the-provider) below). Once the first signed release is published, the [Registry listing](https://registry.terraform.io/providers/rafftechnologies/raff/latest/docs) will be the canonical source.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/install) 1.0+
- [Go](https://go.dev/doc/install) 1.25+ (to build the provider plugin)

## Building The Provider

Clone the repository:

```bash
git clone git@github.com:RaffTechnologies/terraform-provider-raff.git
cd terraform-provider-raff
```

Build the provider:

```bash
go build -o terraform-provider-raff
```

To use a locally-built provider with Terraform, add a `dev_overrides` block to your `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "rafftechnologies/raff" = "/absolute/path/to/terraform-provider-raff"
  }
  direct {}
}
```

## Using the provider

Once the first release is published to the Terraform Registry, declare the provider in your config and run `terraform init`:

```hcl
terraform {
  required_providers {
    raff = {
      source  = "rafftechnologies/raff"
      version = "~> 0.1"
    }
  }
}

provider "raff" {
  api_key    = var.raff_api_key       # or RAFF_API_KEY env var
  project_id = var.raff_project_id    # or RAFF_PROJECT_ID env var
}

resource "raff_vm" "web" {
  name        = "web-01"
  template_id = "5ac21891-32e6-41ce-8a93-b5d6ab708b0d"
  pricing_id  = 3
  region      = "us-east"
  ssh_keys    = ["ssh-ed25519 AAAA... user@host"]
}
```

The provider supports five resources (`raff_project`, `raff_vm`, `raff_vpc`, `raff_ip`, `raff_security_group`) and singular + plural data sources for each. See [internal/provider/](internal/provider/) for the full schema until the Registry docs are live.

## Developing the Provider

See [CONTRIBUTING.md](CONTRIBUTING.md) for information about contributing to this project.

## Documentation

- **Provider docs** — [registry.terraform.io/providers/rafftechnologies/raff/latest/docs](https://registry.terraform.io/providers/rafftechnologies/raff/latest/docs)
- **API reference** — [docs.rafftechnologies.com](https://docs.rafftechnologies.com)
- **Dashboard** — [rafftechnologies.com](https://rafftechnologies.com)
- **Releases / changelog** — [github.com/RaffTechnologies/terraform-provider-raff/releases](https://github.com/RaffTechnologies/terraform-provider-raff/releases)

## Related projects

- [raff-go](https://github.com/RaffTechnologies/raff-go) — official Go SDK that powers this provider
- [raff-cli](https://github.com/RaffTechnologies/raff-cli) — official command-line interface

## License

[MIT](LICENSE)

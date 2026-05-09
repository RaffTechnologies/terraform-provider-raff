# Raff Terraform Provider

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

- Documentation: [registry.terraform.io/providers/rafftechnologies/raff/latest/docs](https://registry.terraform.io/providers/rafftechnologies/raff/latest/docs)

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

See the [Raff Provider documentation](https://registry.terraform.io/providers/rafftechnologies/raff/latest/docs) to get started using the Raff provider.

Quick example:

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
  api_key    = var.raff_api_key       # or RAFF_API_KEY
  project_id = var.raff_project_id    # or RAFF_PROJECT_ID
}

resource "raff_vm" "web" {
  name        = "web-01"
  template_id = "5ac21891-32e6-41ce-8a93-b5d6ab708b0d"
  pricing_id  = 3
  region      = "us-east"
  ssh_keys    = ["ssh-ed25519 AAAA... user@host"]
}
```

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

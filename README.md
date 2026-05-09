# Terraform Provider for Raff Cloud

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Official Terraform provider for managing [Raff Cloud](https://rafftechnologies.com) infrastructure. Built on top of [raff-go](https://github.com/RaffTechnologies/raff-go).

## Requirements

- Terraform 1.0+
- A Raff API key (`raff_pub_xxx`) — generate one in the dashboard under **Team & Projects → API Keys**.

## Usage

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
  api_key    = var.raff_api_key       # or set RAFF_API_KEY
  project_id = var.raff_project_id    # or set RAFF_PROJECT_ID
}

resource "raff_project" "prod" {
  name           = "production"
  description    = "Customer-facing workloads"
  default_region = "us-east"
}

resource "raff_vpc" "prod" {
  name   = "prod-net"
  cidr   = "10.0.0.0/20"
  region = "us-east"
}

resource "raff_security_group" "web" {
  name        = "web"
  description = "Public web tier"

  rule {
    rule_type = "inbound"
    protocol  = "TCP"
    range     = "80,443"
  }
  rule {
    rule_type = "inbound"
    protocol  = "TCP"
    range     = "22"
  }
  rule {
    rule_type = "outbound"
    protocol  = "ALL"
  }
}

resource "raff_vm" "web" {
  name        = "web-01"
  template_id = "5ac21891-32e6-41ce-8a93-b5d6ab708b0d"
  pricing_id  = 3
  region      = "us-east"
  ssh_keys    = ["ssh-ed25519 AAAA... user@host"]
  tags        = ["env:prod", "tier:web"]
}
```

## Authentication

Set credentials via the provider block or environment variables:

| Variable | Description |
|----------|-------------|
| `RAFF_API_KEY` | API key (required) |
| `RAFF_API_URL` | API base URL (default `https://api.rafftechnologies.com`) |
| `RAFF_PROJECT_ID` | Default project for project-scoped resources |

Environment variables take effect when the matching provider attribute is omitted.

## Resources

| Resource | Operations |
|----------|-----------|
| `raff_project` | Manages a project. Updatable: name, description, default_region. |
| `raff_vm` | Manages a VM. Updatable: name (rename), pricing_id (resize), tags. Storage / VPC / network attributes force replacement. |
| `raff_vpc` | Manages a VPC. Updatable: name, description. CIDR and region force replacement. |
| `raff_ip` | Reserves a floating IP. Type, region, billing_period force replacement. |
| `raff_security_group` | Manages a security group. Updatable: name, description, rules. Rule updates replace the full set. |

## Data Sources

Both a singular (by ID) and plural (list with optional filters) data source are available for every resource type:

| Data source | Filters (plural) |
|-------------|-----------------|
| `raff_project` / `raff_projects` | — |
| `raff_vm` / `raff_vms` | `region`, `status` |
| `raff_vpc` / `raff_vpcs` | — |
| `raff_ip` / `raff_ips` | `status`, `reserved` |
| `raff_security_group` / `raff_security_groups` | — |

Example:

```hcl
data "raff_vms" "running" {
  status = "active"
  region = "us-east"
}

output "running_vm_count" {
  value = length(data.raff_vms.running.vms)
}
```

## Versioning

Semantic versioning. v0.x is allowed to introduce breaking schema changes; v1.0.0+ implies a stable resource schema. See [CHANGELOG](https://github.com/RaffTechnologies/terraform-provider-raff/releases) for per-release details.

## Development

### Build locally

```bash
go build -o terraform-provider-raff
```

### Use a locally-built provider

```bash
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/rafftechnologies/raff/0.0.1/$(go env GOOS)_$(go env GOARCH)
cp terraform-provider-raff ~/.terraform.d/plugins/registry.terraform.io/rafftechnologies/raff/0.0.1/$(go env GOOS)_$(go env GOARCH)/

cat > ~/.terraformrc <<EOF
provider_installation {
  dev_overrides {
    "rafftechnologies/raff" = "$HOME/.terraform.d/plugins/registry.terraform.io/rafftechnologies/raff/0.0.1/$(go env GOOS)_$(go env GOARCH)"
  }
  direct {}
}
EOF
```

### Tests

```bash
go test ./...
go vet ./...
```

## Releases

Releases are GPG-signed and published to the [Terraform Registry](https://registry.terraform.io/providers/rafftechnologies/raff). Maintainers — see [the release workflow](.github/workflows/release.yml) and the GPG key setup notes inside.

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

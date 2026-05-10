# Raff Terraform Provider

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Terraform provider for [Raff](https://rafftechnologies.com), built on [raff-go](https://github.com/RaffTechnologies/raff-go).

> **14 resources, 25 data sources** covering the full Raff public API: compute (VMs, volumes, snapshots, backups, backup schedules), networking (VPCs, IPs, security groups), identity (projects, members, roles, API keys, SSH keys), and read-only catalogs (regions, templates, pricing). Built on [raff-go](https://github.com/RaffTechnologies/raff-go).

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

resource "raff_ssh_key" "laptop" {
  name       = "laptop"
  public_key = file("~/.ssh/id_ed25519.pub")
}

data "raff_templates" "ubuntu" {
  category = "linux"
  vm_type  = "standard"
}

data "raff_vm_pricing" "standard_us" {
  region  = "us-east"
  vm_type = "standard"
}

resource "raff_vm" "web" {
  name        = "web-01"
  template_id = data.raff_templates.ubuntu.templates[0].id
  pricing_id  = 9   # standard 2vCPU/4GB/50GB ($4.99/mo) — see data.raff_vm_pricing.standard_us
  region      = "us-east"
}

resource "raff_volume" "data" {
  name        = "data-vol"
  size        = 100
  volume_type = "nvme"
  region      = "us-east"
  vm_id       = raff_vm.web.id
}

resource "raff_backup_schedule" "nightly" {
  vm_id      = raff_vm.web.id
  frequency  = "daily"
  time       = "03:00"
  keep_count = 14
}
```

## Resources & Data Sources

**14 resources, 25 data sources.** Full inventory:

### Resources

| Category | Resource | Notes |
|---|---|---|
| Compute | `raff_vm` | VM lifecycle |
| Compute | `raff_volume` | Block storage volume + attachment |
| Compute | `raff_snapshot` | VM-disk or volume snapshot |
| Compute | `raff_backup` | One-shot VM backup. ForceNew on every field — backups are immutable once captured. |
| Compute | `raff_backup_schedule` | Recurring backup policy. Manages its own retention; don't manage backups it creates via `raff_backup`. |
| Networking | `raff_vpc` | Virtual private cloud |
| Networking | `raff_ip` | Reserved floating IP |
| Networking | `raff_security_group` | Firewall ruleset |
| Identity | `raff_project` | Project |
| Identity | `raff_project_member` | User/API-key in a project |
| Identity | `raff_member` | Account-level membership / invite |
| Identity | `raff_role` | Custom IAM role |
| Identity | `raff_api_key` | API key (secret stored in tfstate — lock it down) |
| Identity | `raff_ssh_key` | SSH public key |

### Data sources

Singular (`raff_<name>` by ID) and plural (`raff_<name>s` for listing) for every resource above (where the API exposes both), **plus** read-only catalog lookups with no resource counterpart:

| Data source | Purpose |
|---|---|
| `raff_regions` | Available datacenter regions |
| `raff_templates` | OS templates (filter: category, vm_type, region) |
| `raff_vm_pricing` | VM size catalog with `pricing_id` to use in `raff_vm` |
| `raff_volume_pricing` | Per-GB volume storage rate |
| `raff_backup_pricing` | Per-GB backup storage rate |
| `raff_snapshot_pricing` | Per-GB snapshot storage rate |
| `raff_ip_pricing` | Floating IP rates by family (ipv4 / ipv6) |

### Not exposed (intentional)

- **Invitations** — transient by nature (accepted/cancelled means it's gone). Terraform managing "this invitation should exist" loops. Use `raff_member { email = ... }` for the equivalent declarative shape — the underlying invitation is auto-created and cleaned up.

### When to use `raff_backup` vs `raff_backup_schedule`

- **`raff_backup`** — for capturing a single backup before a known event (a deployment, a manual checkpoint). The backup is owned by the Terraform config that created it; removing the resource block deletes the backup.
- **`raff_backup_schedule`** — for ongoing protection. The schedule manages its own retention (`keep_count`); auto-pruned backups don't appear in any TF state. **Don't** also manage schedule-created backups via `raff_backup` — you'll fight the retention engine.

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

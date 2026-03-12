# Terraform Provider for Raff Cloud

Terraform provider for managing [Raff Cloud](https://rafftechnologies.com) infrastructure.

## Usage

```hcl
terraform {
  required_providers {
    raff = {
      source = "rafftechnologies/raff"
    }
  }
}

provider "raff" {
  api_key = "raff_pub_xxx"   # or set RAFF_API_KEY env var
}

resource "raff_project" "example" {
  name           = "my-project"
  description    = "Production workloads"
  default_region = "us-east"
}
```

## Authentication

Set your API key via provider config or environment variable:

```bash
export RAFF_API_KEY="raff_pub_xxx"
```

## Resources

- `raff_project` — Manage projects

## Development

```bash
# Build
go build -o terraform-provider-raff

# Install locally for testing
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/rafftechnologies/raff/0.1.0/darwin_arm64
cp terraform-provider-raff ~/.terraform.d/plugins/registry.terraform.io/rafftechnologies/raff/0.1.0/darwin_arm64/

# Test
cd examples/ && terraform init && terraform plan
```

## License

MIT

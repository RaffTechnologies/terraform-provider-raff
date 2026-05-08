# terraform-provider-raff — Terraform Provider

Terraform provider for managing Raff cloud resources. Built on `raff-go` client library.

## Critical Rules

1. **Built on raff-go** — never call the API directly. All API calls go through the `raff-go` client.
2. **Public API only** — schema must match `docs/api-reference/openapi.yaml` exactly.
3. **Never expose admin/internal fields** — no admin endpoints, no `X-Account-ID`, no internal-only attributes.
4. **Sync order**: spec → raff-go → raff-cli → **terraform-provider-raff**
5. **terraform-plugin-framework** — use the modern SDK, not the legacy SDKv2.

## Structure

```
main.go                 # Provider entry point
internal/
└── provider/           # Resources and datasources
```

## Patterns

- **Resources**: `raff_vm`, `raff_project`, `raff_volume`, etc.
- **Datasources**: `raff_vms`, `raff_projects` (read-only lists)
- Each resource maps to a public API endpoint via raff-go
- Schema attributes must match the public API response fields exactly

## Quick Commands

```bash
go build ./...     # Build
go test ./...      # Test
```

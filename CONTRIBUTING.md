# Contributing to terraform-provider-raff

Thank you for your interest in contributing!

## Reporting bugs

Open a [GitHub issue](https://github.com/RaffTechnologies/terraform-provider-raff/issues) with:
- Terraform version (`terraform -v`)
- Provider version
- A minimal reproducible config
- Expected vs actual behavior
- Any error output (with sensitive data redacted)

## Development workflow

1. Fork the repo and create a feature branch.
2. Install dependencies: `go mod download`
3. Build: `go build -o terraform-provider-raff`
4. Run tests + vet:
   ```bash
   go test ./...
   go vet ./...
   ```
5. Test against a real Raff account using a local `dev_overrides` block in `~/.terraformrc` (see the [README](README.md#building-the-provider)).
6. Open a PR with a clear description of what changed and why.

## Code style

- Match existing patterns in [internal/provider/](internal/provider/). Each resource is `resource_<name>.go`; each data source is `data_source_<name>.go`.
- All API calls go through [raff-go](https://github.com/RaffTechnologies/raff-go). Never make direct HTTP calls.
- Schema attributes must match the public API spec exactly. Never expose admin/internal fields.

## Adding a new resource

1. Verify the operations exist in [raff-go](https://github.com/RaffTechnologies/raff-go) first. If they don't, add them to raff-go before extending the provider.
2. Mirror an existing resource (e.g. [resource_vpc.go](internal/provider/resource_vpc.go)) for shape: schema, Create / Read / Update / Delete, plus a flatten helper.
3. Add a singular and plural data source ([data_source_vpc.go](internal/provider/data_source_vpc.go) is a good template).
4. Register in [provider.go](internal/provider/provider.go) under `ResourcesMap` and `DataSourcesMap`.
5. Confirm `go build ./...` and `go vet ./...` pass.

## Releasing

Releases are cut by maintainers. See [.github/workflows/release.yml](.github/workflows/release.yml) for the GoReleaser flow. The Terraform Registry requires GPG-signed checksums; the workflow imports the key from `GPG_PRIVATE_KEY` and `PASSPHRASE` repository secrets.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).

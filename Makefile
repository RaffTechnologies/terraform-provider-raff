.PHONY: build test vet lint sync docs clean

build:
	go build ./...

# Regenerate Registry docs from schema, then apply sidebar subcategories.
# tfplugindocs emits subcategory: "" by default — the script fills it in
# so the Registry groups resources by product area (VMs, Volumes, etc.).
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate
	./scripts/set-doc-subcategories.sh

test:
	go test ./...

vet:
	go vet ./...

lint:
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed; skipping"

# Build, vet, test against the current raff-go.
sync: build vet test

clean:
	go clean ./...

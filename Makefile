.PHONY: build test vet lint sync clean

build:
	go build ./...

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

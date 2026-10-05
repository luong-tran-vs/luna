GOLANGCI_LINT_IMAGE := golangci/golangci-lint:v2.14.0

.PHONY: run test lint build

run:
	go run ./cmd/server

test:
	go test ./...

# golangci-lint runs in Docker so it does not need to be installed locally.
# Named volumes cache Go modules and lint results between runs (offline after the first run).
lint:
	docker run --rm -v "$(CURDIR):/app" -w /app \
		-v luna-go-mod:/go/pkg/mod -v luna-golangci-cache:/root/.cache \
		$(GOLANGCI_LINT_IMAGE) golangci-lint run ./...

build:
	CGO_ENABLED=0 go build -trimpath -o bin/server ./cmd/server

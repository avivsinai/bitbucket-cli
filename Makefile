.PHONY: build fmt lint test tidy sbom release

build:
	go build ./cmd/bkt

fmt:
	go fmt ./...

lint:
	golangci-lint run

test:
	go test ./...

tidy:
	go mod tidy

sbom:
	@if ! command -v syft >/dev/null 2>&1; then \
		echo "syft not installed; install from https://github.com/anchore/syft" >&2; \
		exit 1; \
	fi
	syft dir:. -o cyclonedx-json=sbom.cdx.json

release:
	goreleaser release --clean

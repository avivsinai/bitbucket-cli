.PHONY: build fmt lint test tidy

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


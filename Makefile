.PHONY: build test vet fmt fmt-check tidy install help

## build: Compile the provider binary
build:
	go build ./...

## test: Run unit tests (schema validation, etc.)
test:
	go test -v -race ./...

## vet: Run go vet
vet:
	go vet ./...

## fmt: Format all Go files
fmt:
	gofmt -s -w .

## fmt-check: Fail if any Go file is not gofmt-formatted
fmt-check:
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not gofmt-formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

## tidy: Tidy and verify go.mod / go.sum
tidy:
	go mod tidy
	go mod verify

## install: Build and install the provider binary to GOBIN
install:
	go install .

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'

.DEFAULT_GOAL := build

.PHONY: build test testacc sanity sanity-tf vet fmt fmt-check lint tidy install docs docs-check help

## build: Compile the provider binary
build:
	go build ./...

## test: Run unit tests (schema validation, etc.)
test:
	go test -v -race ./...

## testacc: Run acceptance tests against a live Graphiant tenant (requires
## TF_ACC=1 and GRAPHIANT_ACCESS_TOKEN, or GRAPHIANT_USERNAME+GRAPHIANT_PASSWORD)
testacc:
	TF_ACC=1 go test -v -timeout 30m ./...

## sanity: Terraform-independent smoke test — log in and list the edge summary
## (requires GRAPHIANT_ACCESS_TOKEN, or GRAPHIANT_USERNAME+GRAPHIANT_PASSWORD)
sanity:
	go run ./cmd/sanity

## sanity-tf: Same check as `sanity`, but through the real provider binary and
## the real Terraform plugin protocol via a throwaway dev override (requires
## GRAPHIANT_ACCESS_TOKEN, or GRAPHIANT_USERNAME+GRAPHIANT_PASSWORD, and terraform on PATH)
sanity-tf:
	./scripts/terraform-sanity.sh

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

## lint: Run golangci-lint (same config as CI)
lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...

## tidy: Tidy and verify go.mod / go.sum
tidy:
	go mod tidy
	go mod verify

## install: Build and install the provider binary to GOBIN
install:
	go install .

## docs: Regenerate docs/ from examples/ and schema descriptions (tfplugindocs)
docs:
	go tool tfplugindocs generate

## docs-check: Fail if docs/ is out of date with examples/ and schema descriptions
docs-check:
	go tool tfplugindocs generate
	@if [ -n "$$(git status --porcelain docs/)" ]; then \
		echo "docs/ is out of date — run 'make docs' and commit the result:"; \
		git status --porcelain docs/; \
		exit 1; \
	fi

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'

.DEFAULT_GOAL := build

# Contributing to the Graphiant Terraform Provider

Thank you for your interest in contributing!

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork:**
   ```bash
   git clone https://github.com/Graphiant-Inc/terraform-provider-graphiant.git
   cd terraform-provider-graphiant
   ```
3. **Set up your development environment:**
   ```bash
   # Ensure Go 1.25+ is installed (enforced by go.mod)
   go version

   # Download dependencies and verify
   make tidy
   ```

## Development Workflow

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** and ensure they pass local checks:
   ```bash
   make build       # compile
   make test        # go test -v -race ./...
   make vet         # go vet ./...
   make fmt-check    # fail if any file isn't gofmt-formatted
   make lint        # golangci-lint, same config as CI
   make tidy        # go mod tidy && go mod verify

   # Or individually:
   gofmt -s -w .
   go vet ./...
   go test -v -race ./...
   ```

3. **Commit with clear messages** and push, then open a pull request.

## Project Layout

- `main.go` — provider entrypoint (`providerserver.Serve`)
- `internal/provider/provider.go` — provider schema, configuration, and the
  `Resources()` / `DataSources()` registries
- `internal/provider/client.go` — wraps `graphiant-sdk-go`'s `*APIClient`
  together with the resolved bearer token (`gClient`)
- `internal/provider/*_resource.go` — one file per managed resource
  (`site`, `group`, `user`)
- `internal/provider/*_data_source.go` — one file per data source
  (singular = lookup by ID, plural = list all)
- `internal/provider/util.go`, `location.go`, `permissions.go` — shared
  conversion helpers and nested-attribute schemas reused across resources

## Adding a New Resource or Data Source

This provider is a hand-written wrapper around
[`graphiant-sdk-go`](https://github.com/Graphiant-Inc/graphiant-sdk-go), not
generated code. When adding a new resource or data source:

1. Check `graphiant-sdk-go`'s `docs/DefaultAPI.md` for the relevant endpoints
   and `docs/<Model>.md` for the request/response shapes. Note that despite
   what the generated docs say, list-typed model fields (`[]string`,
   `[]int64`, ...) are **not** pointers in the actual generated structs —
   only scalar fields are; verify against `model_*.go` if in doubt.
2. Follow the existing pattern: a `tfsdk`-tagged model struct, a `Schema()`
   method, and `expand*`/`flatten*` helpers converting between
   `types.X` and the SDK struct (see `util.go` for the shared helpers).
3. If the API has no get-by-id endpoint, follow `findSite`/`findGroup`'s
   pattern of listing and filtering client-side in `Read`.
4. If a write endpoint doesn't return the created/updated object (as with
   `V1GroupsPut`/`V1UsersPut`), read it back afterward as `Create`/`Update`
   already do for groups and users.
5. Register the new `New*Resource`/`New*DataSource` constructor in
   `provider.go`'s `Resources()`/`DataSources()`.
6. `TestResourceSchemas`/`TestDataSourceSchemas` iterate the registries in
   `provider.go`, so the new resource/data source is covered automatically —
   no changes needed there. Do add dedicated tests if its expand/flatten
   logic is non-trivial.

## Testing

### Unit tests

```bash
go test ./...
```

`internal/provider/provider_test.go` walks every registered resource and
data source and validates its schema with the framework's
`Schema.ValidateImplementation`. Add unit tests for any new `expand`/`flatten`
helper that does non-trivial conversion.

### Acceptance tests

This provider does not yet have `terraform-plugin-testing`-based acceptance
tests that exercise a live Graphiant API. If you add them, follow the
upstream convention of gating them behind `TF_ACC=1` and the same
`GRAPHIANT_ACCESS_TOKEN` / `GRAPHIANT_USERNAME` + `GRAPHIANT_PASSWORD`
environment variables used by [`graphiant-sdk-go`](https://github.com/Graphiant-Inc/graphiant-sdk-go/blob/main/CONTRIBUTING.md#environment-variables-for-tests),
so tests skip gracefully when credentials aren't configured.

### Manual testing against a real Graphiant tenant

```bash
go build -o terraform-provider-graphiant .
```

Point Terraform at your local build with a
[dev override](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers)
in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "Graphiant-Inc/graphiant" = "/absolute/path/to/terraform-provider-graphiant"
  }
  direct {}
}
```

With the override in place, `terraform plan`/`apply` in a directory with a
`graphiant` provider block will use your local build instead of the registry.

## Code Standards

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines.
- Use `gofmt` for formatting (checked via `make fmt-check`).
- Keep resource/data-source files symmetric with their siblings (e.g.
  `site_resource.go` and `site_data_source.go` should model the same fields
  the same way) so the provider stays predictable to extend.
- Don't add attributes, resources, or data sources speculatively — only for
  API surface the provider actually needs to expose.
- Handle errors explicitly and surface them via `resp.Diagnostics`, never by
  panicking or silently dropping them. Use `apiErrorDetail(err)` (`util.go`)
  rather than `err.Error()` for SDK call failures — `graphiant-sdk-go`'s
  `GenericOpenAPIError.Error()` only returns the HTTP status line (e.g.
  `"400 Bad Request"`); `apiErrorDetail` appends the actual response body.
- Close every SDK response body with `defer closeBody(httpRes)` (`util.go`),
  not `defer httpRes.Body.Close()` directly — the latter trips
  `golangci-lint`'s `errcheck` and there's no reason to inline the nil check
  at every call site.
- On a **resource** (not data source) schema, add a plan modifier to every
  `Computed`-only attribute unless its value is a deterministic side effect
  of that resource's own `Update`:
  - `UseStateForUnknown()` for attributes that don't change as a side effect
    of updating other fields (most of them — e.g. a site's `tags`, a user's
    `verified`). Without it, the attribute shows `(known after apply)` on
    every single plan, not just when something relevant actually changed.
  - No modifier for attributes that genuinely change on every update (e.g.
    a site's `updated_at`) — they should show as unknown when the resource
    is modified.
- Add `tflog.Debug`/`tflog.Trace` calls at the start and end of each CRUD
  method (and at the start of data source `Read`s) so `TF_LOG=DEBUG` gives
  operators something to go on. Keep it to key identifying fields (id,
  email, name) — see any existing resource file for the pattern.
- Prefer schema-level validators (`terraform-plugin-framework-validators`)
  over deep-diving into API semantics you don't have confirmed — e.g.
  `stringvalidator.LengthAtLeast(1)` for required strings, or
  `resourcevalidator.RequiredTogether` for fields that only make sense
  paired. Don't invent enum constraints (e.g. permission values) unless
  the API docs actually document the closed set — guessed constraints
  reject valid configs.
- Provider-level (not resource-level) auth fields (`access_token`,
  `username`, `password`) each independently fall back to an environment
  variable. Don't add cross-field `ConfigValidators` for these — a
  validator can't see resolved env values, so e.g. requiring
  `username`/`password` together at the schema level would break the
  common pattern of a non-secret username in `.tf` and a secret password
  from `GRAPHIANT_PASSWORD`.

## Pull Request Checklist

- [ ] `make build` passes
- [ ] `make test` passes
- [ ] `make vet` passes
- [ ] `make fmt-check` passes
- [ ] `make lint` passes
- [ ] `make tidy` leaves `go.mod` and `go.sum` unchanged
- [ ] New/changed resources and data sources have matching schema coverage
      (they'll be picked up automatically by `TestResourceSchemas`/
      `TestDataSourceSchemas` once registered in `provider.go`)
- [ ] Commit messages are clear

## Continuous Integration

Every pull request and push to `main` runs:

- **[test.yml](.github/workflows/test.yml)** — `go build`, `go vet`, `go test -race`
- **[lint.yml](.github/workflows/lint.yml)** — `golangci-lint` and a `gofmt` check

## Releasing

Pushing a tag matching `v*` (e.g. `v0.1.0`) triggers
**[release.yml](.github/workflows/release.yml)**, which runs
[GoReleaser](https://goreleaser.com) per [`.goreleaser.yml`](.goreleaser.yml)
to build cross-platform binaries, sign the checksum file with GPG (required
for the Terraform Registry), and publish a GitHub release.

This requires two repository secrets that are not configured by default:

- `GPG_PRIVATE_KEY` — an ASCII-armored GPG private key whose public key is
  registered with the Terraform Registry's publisher settings for this
  provider.
- `PASSPHRASE` — the passphrase for that key.

`GITHUB_TOKEN` is provided automatically by GitHub Actions. See HashiCorp's
[provider publishing guide](https://developer.hashicorp.com/terraform/registry/providers/publishing)
for how to generate and register the GPG key before cutting the first
release.

## Additional Resources

- [Terraform Plugin Framework docs](https://developer.hashicorp.com/terraform/plugin/framework)
- [graphiant-sdk-go](https://github.com/Graphiant-Inc/graphiant-sdk-go)
- [Go Documentation](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)

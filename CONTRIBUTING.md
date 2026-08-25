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
  (`site`, `group`, `user`, `site_list`, `content_filter`, `app_list`,
  `custom_app`)
- `internal/provider/*_data_source.go` — one file per data source
  (singular = lookup by ID, plural = list all)
- `internal/provider/util.go`, `location.go`, `permissions.go` — shared
  conversion helpers reused across resources
- `internal/provider/generated/` — schemas and model structs generated from
  `api/graphiant_api_docs_v26.7.0.json` via `make generate-schemas`
  (**do not hand-edit**, headed `DO NOT EDIT`); see
  [Adding a New Resource or Data Source](#adding-a-new-resource-or-data-source)
- `internal/provider/acctest_test.go` — shared acceptance-test helpers
  (`testAccProtoV6ProviderFactories`, `testAccPreCheck`)
- `internal/provider/*_test.go` (`TestAcc*` functions) — acceptance tests
  against a live Graphiant API; see [Acceptance tests](#acceptance-tests)
- `examples/` — runnable `.tf` config per resource/data source, in the layout
  `terraform-plugin-docs`/the Terraform Registry expect
- `docs/` — generated attribute reference; **do not hand-edit**, run
  `make docs` instead (see [Documentation](#documentation) below)

## API Coverage

`graphiant-sdk-go` exposes roughly 525 endpoints; this provider deliberately
covers a subset of them. A resource is only added for an endpoint (or small
group of endpoints) that supports full create/read/delete — ideally
create/read/update/delete — lifecycle management of a persistent object.
Endpoints that are actions (e.g. resetting an IPsec session, approving a
device return), analytics/telemetry queries (bandwidth trackers, top
talkers, flow reports), session/account self-service (login, MFA, SAML,
password), or things like AI assistant conversations and audit/activity logs
are **not** modeled as resources — forcing them into Terraform's
create/read/update/delete resource model produces broken or misleading
resources (e.g. a "resource" with no real update or delete semantics). This
follows HashiCorp's own
[provider design principles](https://developer.hashicorp.com/terraform/plugin/best-practices/hashicorp-provider-design-principles)
("single API object per resource") and its guidance that ephemeral resources
and provider functions — not resources — exist for things that aren't
long-lived state.

**Covered today:**

- IAM: `graphiant_site`, `graphiant_group`, `graphiant_user`
- Devices: read-only (`graphiant_device`/`graphiant_devices`) — this
  provider does not manage device network configuration
- Global catalog objects: `graphiant_site_list`, `graphiant_content_filter`,
  `graphiant_app_list`, `graphiant_custom_app`

**Confirmed full-CRUD candidates not yet implemented** (verified against the
SDK's actual request/response models, not just endpoint names — see git
history/PR discussion for the audit): `pvif` (physical/virtual interfaces)
and `gateways` (note: gateway update/delete take the id in the request body
rather than the URL path — a shape worth designing carefully). Larger,
multi-object domains needing more design work before implementation:
`extranets`/`extranet/b2b/*` (a multi-object B2B workflow: customers,
consumers, producers, matches) and `enterprises`/`enterprise` (multi-tenant
admin). Device network configuration (interfaces, circuits, routing) is
configured via a single monolithic `PUT /v1/devices/{deviceId}/config`
rather than per-object endpoints, which changes the resource design
significantly and needs that request body's model read in full before any
schema is designed.

**Ruled out** (endpoint exists but doesn't support full CRUD in the current
API — e.g. create-only with no way to read back or delete): global
`prefix-sets`, `routing-policies`, `traffic-policies`, `ntps`, `snmps`,
`syslogs`, `ipfix` (all POST-only); `ipsec-profile` (GET-only, a data-source
candidate at most if ever needed).

When proposing a new resource, check the actual generated model structs
(`model_*.go` in `graphiant-sdk-go`) for the relevant endpoints before
assuming an endpoint is CRUD-capable — endpoint *names* are not a reliable
signal (see `pvif`/`gateways` above for confirmed-good shapes vs. the
POST-only domains above that look like resources but aren't).

## Adding a New Resource or Data Source

Schemas and model structs are **generated from the OpenAPI spec**
(`api/graphiant_api_docs_v26.7.0.json`) via `make generate-schemas`
(`api/generate.sh`) — see `api/generator_config.yml` for the path/method
mapping per resource/data source. Everything under
`internal/provider/generated/` is regenerated output; don't hand-edit it.
CRUD logic still hand-calls `graphiant-sdk-go` directly (the generator
doesn't produce a client or any Create/Read/Update/Delete code at all, only
schema + model types) — see `internal/provider/*_resource.go` for the
pattern. When adding a new resource or data source:

1. Add its `create`/`read`/`update`/`delete` (or `read`-only, for a data
   source) path/method mapping to `api/generator_config.yml`. If the
   request/response body wraps the real fields in an envelope object, or the
   endpoint has no genuine single-object read (see `api/augment_spec.py`'s
   module docstring for why this comes up and how it's handled), extend
   `api/augment_spec.py` rather than fighting the generator — it produces a
   codegen-only derived spec used solely as generator input.
2. Run `make generate-schemas`. Diff the result — this toolchain has known
   rough edges (see comments in `api/patch_ir.py` and
   `api/dedupe_generated.py`), so treat a clean run as something to verify,
   not assume. In particular: `generator_config.yml` can't express plan
   modifiers, `RequiresReplace`, or computed/optional overrides — add those
   to `api/patch_ir.py` instead, which hand-patches the intermediate
   representation between the OpenAPI→IR and IR→Go steps.
3. Write the resource/data source Go file: a `Schema()` that calls into the
   generated package and layers any hand-patched attributes (server-derived
   counters not present in the generated response, plan modifiers/validators
   not expressible via config) on top — see `app_list_resource.go` for the
   reference pattern — plus `Create`/`Read`/`Update`/`Delete` calling
   `graphiant-sdk-go` as normal. The framework's struct reflection works off
   the raw `tftypes` value, so a plain `tfsdk`-tagged model struct works
   fine as the destination regardless of the generated schema's `CustomType`
   wrapper — you don't need the generated nested-object value types unless
   you want their typed accessors.
4. If the API has no get-by-id endpoint, follow `findSite`/`findGroup`'s
   pattern of listing and filtering client-side in `Read`.
5. If a write endpoint doesn't return the created/updated object (as with
   `V1GroupsPut`/`V1UsersPut`), read it back afterward as `Create`/`Update`
   already do for groups and users.
6. Register the new `New*Resource`/`New*DataSource` constructor in
   `provider.go`'s `Resources()`/`DataSources()`.
7. `TestResourceSchemas`/`TestDataSourceSchemas` iterate the registries in
   `provider.go`, so the new resource/data source is covered automatically —
   no changes needed there. Do add dedicated tests if its expand/flatten
   logic is non-trivial.
8. Add an acceptance test (see [Acceptance tests](#acceptance-tests) below)
   and an example config:
   - Resource: `examples/resources/graphiant_<name>/resource.tf`, plus
     `import.sh` if it implements `ResourceWithImportState`.
   - Data source: `examples/data-sources/graphiant_<name>/data-source.tf`.
9. Run `make docs` to regenerate `docs/` from the new schema and example —
   CI's `docs` job (`lint.yml`) fails the PR otherwise.

Exception: `graphiant_device`/`graphiant_devices` are hand-written, not
generated — the real API schema for a device is a ~45-field, deeply-nested
object that reuses attribute names (e.g. `rules`, `bgp`) at different
nesting depths, which hits an unresolved upstream bug where
`tfplugingen-framework` emits colliding duplicate Go types for them (see
`api/generator_config.yml`'s comment for links). Don't attempt to bring
these into codegen without checking whether that upstream issue has been
fixed.

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

`internal/provider/*_test.go` has `terraform-plugin-testing`-based acceptance
tests (`TestAccSiteResource`, `TestAccGroupResource`, `TestAccUserResource`,
`TestAccSiteListResource`, `TestAccContentFilterResource`,
`TestAccAppListResource`, `TestAccCustomAppResource`,
`TestAccDevicesDataSource`) that exercise a live Graphiant API: full
create → read → update → import cycles for each resource, plus their
associated data sources. They're gated the same way
[`graphiant-sdk-go`](https://github.com/Graphiant-Inc/graphiant-sdk-go/blob/main/CONTRIBUTING.md#environment-variables-for-tests)'s
tests are:

- `resource.Test`'s built-in `TF_ACC=1` requirement (unset → self-skip), and
- `testAccPreCheck` (`internal/provider/acctest_test.go`), which additionally
  self-skips unless `GRAPHIANT_ACCESS_TOKEN`, or both `GRAPHIANT_USERNAME` and
  `GRAPHIANT_PASSWORD`, are set.

Run them locally against a real (ideally non-production) tenant:

```bash
export GRAPHIANT_ACCESS_TOKEN="..."   # or GRAPHIANT_USERNAME + GRAPHIANT_PASSWORD
export GRAPHIANT_API_HOST="https://api.graphiant.com"  # optional, defaults as the provider does
make testacc

# Or a single resource's test:
TF_ACC=1 go test ./internal/provider/ -run TestAccSiteResource -v -timeout 30m
```

Acceptance tests create and delete real objects (named with
`acctest.RandomWithPrefix`, e.g. `tf-acc-site-<random>`) — never point them at
a production tenant you can't afford to have test data appear in
transiently. `TestAccDevicesDataSource` only reads (devices can't be created
by this provider) and doesn't assert a specific count.

**Adding acceptance tests for a new resource:** follow the pattern in
`site_resource_test.go` — a `TestAcc<Name>Resource` function with
`resource.Test`, `PreCheck: func() { testAccPreCheck(t) }`,
`ProtoV6ProviderFactories: testAccProtoV6ProviderFactories`, and steps for
create+read, `ImportState`, and an in-place update. Include the resource's
paired data source(s) in the create step's config and checks, as
`site_resource_test.go` does with `data.graphiant_site`/`data.graphiant_sites`.

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

## Documentation

`docs/` is generated by [`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs)
from two inputs — **never hand-edit files under `docs/`**, they will be
overwritten:

1. The `Description` strings on the provider/resource/data-source `Schema()`s
   in `internal/provider/*.go`.
2. The example `.tf`/`.sh` files under `examples/` (see
   [Adding a New Resource or Data Source](#adding-a-new-resource-or-data-source)
   for the expected layout).

Regenerate after changing either:

```bash
make docs
```

CI's `docs` job (`lint.yml`) runs the same command and fails the PR if it
produces a diff, so `docs/` can't silently drift from the schema or examples.

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
- [ ] New/changed resources and data sources have an example under
      `examples/` and, ideally, an acceptance test (`make testacc` passes
      against a live tenant, or was run by a maintainer who has one)
- [ ] `make docs` leaves `docs/` unchanged (or its diff is included in the PR)
- [ ] Commit messages are clear

## Continuous Integration

Every pull request and push to `main` runs:

- **[test.yml](.github/workflows/test.yml)** — `build` (`go build`), `test`
  (`go vet` + `go test -race`, no credentials needed), and `acceptance`
  (`TestAcc*` against a live tenant if `GRAPHIANT_ACCESS_TOKEN` or
  `GRAPHIANT_USERNAME`+`GRAPHIANT_PASSWORD` are configured as repository
  secrets/variables — otherwise those tests self-skip and the job still
  passes). `acceptance` also runs nightly via a `schedule` trigger.
- **[lint.yml](.github/workflows/lint.yml)** — `golangci-lint`, a `gofmt`
  check, `terraform fmt -check` over `examples/`, and the `docs` drift check
  described above.

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

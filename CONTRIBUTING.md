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
- `internal/provider/provider.go` — provider schema, `Configure` (auth
  resolution), and the `Resources()` / `DataSources()` registries
- `internal/provider/client.go` — `providerData` (the SDK `*APIClient` plus
  the resolved bearer token) and the login/host-resolution helpers
- `internal/provider/util.go` — id conversion, response-body-close, and
  `providerData`-extraction helpers shared across every resource/data source
- `internal/provider/errors.go` — `apiErrorDetail`, which unwraps
  `graphiant_sdk.GenericOpenAPIError` for a diagnostic-friendly message
- `internal/provider/*_resource.go` — one file per managed resource
  (`site`, `user`, `group`, `app_list`, `content_filter`, `custom_app`,
  `site_list`), each self-contained: schema, model struct,
  `Create`/`Read`/`Update`/`Delete`, `ImportState`
- `internal/provider/device_data_source.go` — the one data source
  (`graphiant_device`)
- `internal/provider/provider_test.go` — schema-validation unit tests
- `examples/` — runnable `.tf` config per resource/data source, in the layout
  `terraform-plugin-docs`/the Terraform Registry expect
- `docs/` — generated attribute reference; **do not hand-edit**, run
  `make docs` instead (see [Documentation](#documentation) below)

There is no codegen step. Each resource maps Terraform schema fields to
`graphiant-sdk-go` request/response structs by hand — see
[Adding a New Resource or Data Source](#adding-a-new-resource-or-data-source).

## API Coverage

`graphiant-sdk-go` exposes roughly 1,050 endpoints; this provider
deliberately covers a small subset of them. A resource is only added for an
endpoint (or small group of endpoints) that supports full create/read/delete
— ideally create/read/update/delete — lifecycle management of a persistent
object. Endpoints that are actions (e.g. resetting an IPsec session,
approving a device return), analytics/telemetry queries (bandwidth
trackers, top talkers, flow reports), session/account self-service (login,
MFA, SAML, password), or things like AI assistant conversations and
audit/activity logs are **not** modeled as resources — forcing them into
Terraform's create/read/update/delete resource model produces broken or
misleading resources (e.g. a "resource" with no real update or delete
semantics). This follows HashiCorp's own
[provider design principles](https://developer.hashicorp.com/terraform/plugin/best-practices/hashicorp-provider-design-principles)
("single API object per resource") and its guidance that ephemeral resources
and provider functions — not resources — exist for things that aren't
long-lived state.

**Covered today:**

- IAM: `graphiant_site`, `graphiant_group`, `graphiant_user`
- Devices: read-only (`graphiant_device`) — this provider does not manage
  device network configuration
- Global catalog objects: `graphiant_site_list`, `graphiant_content_filter`,
  `graphiant_app_list`, `graphiant_custom_app`
- Enterprise/MSP: `graphiant_enterprise` (credits/billing are fields on this
  resource — there is no standalone credits resource in the API)
- Gateway service: `graphiant_gateway` (core + single-peer IPsec only —
  cloud-provider gateway types and multi-peer IPsec are not exposed)
- Data exchange: `graphiant_public_vif` (Public VIF), `graphiant_extranet`
  (local/intra-tenant), and the B2B partner workflow — `graphiant_b2b_producer_service`
  → `graphiant_b2b_customer` → `graphiant_b2b_match` → `graphiant_b2b_consumer`
  — built against the newer singular `V1ExtranetB2b*` API generation, not the
  older plural `V1ExtranetsB2b*`/`Peering` one (see git history for why: the
  newer generation is the only one with a peer-to-peer/client-to-server
  distinction)
- Software upgrades: `graphiant_software_rollout` (rollout campaigns).
  Per-device ad hoc upgrade actions (`V1DevicesUpgradeSchedulePut`/
  `CancelPut`) are not exposed — they're fire-and-forget with no persisted
  object to model as a resource
- Device lifecycle actions: `graphiant_device_bringup` (activation),
  `graphiant_device_decommission` (hardware-return workflow, keyed by serial
  number). Both are explicitly action-shaped, not object CRUD — see their
  schema `Description`s for what Delete does and doesn't do
- Data assurance: `graphiant_assurance_global` (SLA assurance config),
  `graphiant_assurance_classified_application` (app classification rules).
  ~20 sibling analytics endpoints in this domain (Enterprisesummary,
  Topology*, Bucket*, Scoredetails, AiAdoptionSummary, etc.) are deliberately
  not exposed — see "API Coverage" below.
- Alerts: `graphiant_alert_integration` (delivery integrations, full CRUD),
  `graphiant_alert_notification` (notification routing config — create
  returns no ID, same list-and-match pattern as `graphiant_enterprise`/
  `graphiant_group`; `rule_id_list` is force-new since the update endpoint
  has no field for it)
- Routing: `graphiant_route_tag` (create+delete only, no update endpoint
  exists; Read only confirms the tag id still exists in the API's recursive
  tag tree — it can't reconstruct level_zero/one/two from that tree, so
  those are preserved from configuration rather than refreshed),
  `graphiant_lan_segment` (create+delete only, same reasoning)
- Device config: `graphiant_device_config` — pushes `maintenance_mode` and,
  on edge devices, the `*_enabled` toggles via `V1DevicesDeviceIdConfigPut`,
  the same endpoint `graphiant-playbooks`' 17 device-level Ansible modules
  use for BGP/interfaces/NAT policy/traffic policy/site-to-site VPN/etc. —
  see "API Coverage" below for why those other domains aren't covered by
  this resource. The PUT is an async job, polled via
  `V1DevicesDeviceIdJobsJobIdGet` until `CompletedAt` is set (`JobState`'s
  valid values are undocumented, so string-matching it would be a guess).
- Read-only data sources covering the read side of edge/site monitoring,
  alerts, troubleshooting, and routing: `graphiant_edges`,
  `graphiant_site_devices` (the closest real proxy for "site health" — see
  below), `graphiant_alert_records`, `graphiant_alert_rules`,
  `graphiant_assurance_flex_algos`, `graphiant_assurance_dnsproxy_entries`
  (read-only rather than a resource — see below), `graphiant_troubleshooting_device`,
  `graphiant_troubleshooting_site`, `graphiant_routing_policy`,
  `graphiant_prefix_set`, `graphiant_domain_categories`, `graphiant_regions`,
  `graphiant_ipsec_profiles`

**Deliberately not implemented:**

- Site health as a real endpoint — it doesn't exist. `V1SitesDetailsGet`'s
  doc comment says "site wide status" but its response model has no status
  field at all (confirmed against the OpenAPI spec, not just the generated
  Go struct — treat doc comments in this SDK as unreliable, `route_tag`'s
  own delete/read endpoints have a similar doc-comment/behavior mismatch,
  confirmed via `localVarHTTPMethod` and the response model instead).
  `graphiant_site_devices` is the closest real proxy (per-device
  maintenance/VRRP state), explicitly documented as such.
- `graphiant_assurance_dnsproxy_entry` as a resource — its delete endpoint
  (`V2AssuranceDeleteDnsproxyEntryDelete`) has no path/query/body parameter
  identifying which entry to delete at all (verified by reading the
  generated request-construction code directly, not just the builder
  methods) — implementing Create without a working Delete would let
  Terraform-managed entries leak permanently. `graphiant_assurance_dnsproxy_entries`
  (a data source) is the honest alternative.
- The ~20 time-windowed data-assurance analytics endpoints — a Terraform
  data source is refreshed on every plan; a report over a caller-supplied
  historical time range produces diffs unrelated to actual infrastructure
  drift, so these don't fit the model regardless of how useful the
  underlying report is.
- Alert rule enable/disable, alert allowlist/mutelist entries, assurance
  user reports, AI adoption approve entries — narrower sub-domains with
  either metrics-drift risk (approve entries mix live usage stats into a
  config object) or report-job semantics, deferred rather than rushed.
- All async diagnostic workflows (ping/traceroute/speedtest/packet-capture/
  debug-archive — each needs a token- or job-id-based polling design this
  provider doesn't attempt yet) and all fire-and-forget diagnostic actions
  (BGP reset, clear ARP, interface reset, reboot, reset IPsec session — no
  result to read back, matches the "don't force actions into the model"
  principle above). Endpoints that are synchronous but still trigger a live
  side effect on every read (ping, OTP passcode generation) are excluded
  too — re-running a live network probe on every `terraform plan` is a bad
  data-source fit regardless of sync/async.
- OSPF RIB, BGP neighbor detail/counters, and BGP/VRF route-count endpoints —
  real and read-only, but narrower, and one response shape
  (`V1DeviceRoutingBgpNbrsDetailsGet`) is unusually thin relative to its
  name per a research pass — left as a follow-up rather than built on an
  unconfirmed assumption.
- `V1GlobalTrafficPoliciesPost` — same read-only lookup-by-id shape as
  prefix-sets/routing-policies, but its request body wasn't fully verified
  before this pass closed — skip rather than guess.

- The other 16 device-config sub-domains behind `V1DevicesDeviceIdConfigPut`
  — BGP, interfaces, LAG, DHCP relay, NTP, OSPFv2, static routes, VRRP,
  MACsec, NAT policy, security policy, traffic policy, site-to-site VPN,
  device system (beyond `region`), edge services (beyond the `*_enabled`
  toggles), prefix/port lists. Each is a large nested map on
  `ManaV2CoreDeviceConfig`/`ManaV2EdgeDeviceConfig` that hasn't been
  individually verified; `graphiant_device_config` only covers the fields
  confirmed to round-trip safely through `ManaV2Device` on read
  (`maintenance_mode` and the edge `*_enabled` booleans) — building any of
  these sub-domains as their own resource needs its own dedicated
  field-by-field research pass, each comparable in size to the B2B partner
  workflow.
- Device "staging" and "deactivation" — not distinct concepts anywhere in
  this SDK (confirmed by exhaustive grep); don't invent an endpoint mapping
  that isn't there.
- The older `V1ExtranetsB2b*`/`Peering` B2B API generation — superseded by
  `V1ExtranetB2b*`, which this provider uses instead.

Everything else — LAN segments, most nested sub-configs on the resources
above (enterprise prefix sets, cloud-provider gateway types, several BGP
neighbor sub-fields — each is called out in its resource's schema
`Description`), and the alert/assurance/diagnostics/routing sub-domains
listed above as deliberately not implemented — is not covered. Before
proposing a new resource, check the actual `model_*.go` structs and
`Api*Execute` return signatures for the relevant endpoints in
`graphiant-sdk-go` — endpoint *names* are not a reliable signal for whether
something is genuinely CRUD-capable (e.g. several `Put`/`Delete` endpoints
return `map[string]interface{}` instead of a typed body, some domains use
four different id-passing conventions across their own four CRUD endpoints —
see `gateway_resource.go` — and a few domains are POST-only with no way to
read back or delete what was created).

## Adding a New Resource or Data Source

There's no codegen step — write the resource file directly against
`graphiant-sdk-go`. Use an existing resource as the template (`site_resource.go`
for a resource with list-scan Read and nested location/route_tag blocks,
`app_list_resource.go` or `custom_app_resource.go` for the common
Post-returns-identifier / Get-by-id-for-full-config shape,
`content_filter_resource.go` for cross-field validators, `group_resource.go`
for a resource that also manages a related sub-collection). When adding one:

1. Read the exact request/response struct fields and `Api*Execute` return
   signature for each endpoint you'll call directly from `graphiant-sdk-go`'s
   `model_*.go` files and `api_default.go` — don't assume field names or
   return arity from the endpoint name; several endpoints in this SDK return
   `map[string]interface{}` instead of a typed response, and some CRUD
   endpoints are noticeably asymmetric (e.g. a resource's update body using a
   different field name than its create body for the same concept, or a
   get-by-id response omitting a field only the list response returns).
2. Define a `tfsdk`-tagged model struct and a `Schema()`. Nested objects the
   API returns/accepts as their own struct become a `SingleNestedBlock` (if
   the API always expects the object embedded, like a site's `location`) or
   a `SingleNestedAttribute`/`ListNestedAttribute` (if it needs to be
   assignable as a value, e.g. inside another list). A `NestedAttributeObject`
   (used by `ListNestedAttribute`) only supports `Attributes`, not `Blocks`.
3. Write `Create`/`Read`/`Update`/`Delete` calling `graphiant-sdk-go`
   directly — chain `.Authorization(pd.token)` onto every request builder
   (there's no client-level "set once" auth in this SDK), and always
   `closeBody(httpResp)` (`util.go`) on the raw response.
4. If the API has no get-by-id endpoint, follow `findSite`'s pattern of
   listing and filtering client-side in `Read`. If a write endpoint doesn't
   return the created/updated object (`V1GroupsPut`/`V1UsersPut`), read it
   back afterward the way `group_resource.go`/`user_resource.go` do. If a
   field is genuinely immutable (no update endpoint touches it — see
   `user_resource.go`, where nearly everything is `RequiresReplace`), mark
   it with `stringplanmodifier.RequiresReplace()` rather than pretending
   `Update` can change it.
5. On a **resource** (not data source) schema, add
   `stringplanmodifier.UseStateForUnknown()` to `id` and any other
   `Computed`-only attribute that doesn't change as a side effect of
   updating other fields — otherwise it shows `(known after apply)` on
   every single plan, not just when something relevant actually changed.
6. Register the new `New*Resource`/`New*DataSource` constructor in
   `provider.go`'s `Resources()`/`DataSources()`.
   `TestResourceSchemas`/`TestDataSourceSchemas` (`provider_test.go`) then
   cover it automatically — no test changes needed unless it has
   non-trivial conversion logic worth a dedicated unit test.
7. Add an example config: `examples/resources/graphiant_<name>/resource.tf`
   (plus `import.sh` if it implements `ResourceWithImportState`), or
   `examples/data-sources/graphiant_<name>/data-source.tf`.
8. Run `make docs` to regenerate `docs/` from the new schema and example —
   CI's `docs` job (`lint.yml`) fails the PR otherwise.
9. Consider adding an acceptance test — see [Testing](#testing) below; this
   provider doesn't have any yet, so there's no existing pattern to copy.

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

Every resource and data source has a `TestAcc<Name>Resource` /
`TestAcc<Name>DataSource` function in `internal/provider/*_test.go`, using the
shared `testAccProtoV6ProviderFactories` / `testAccPreCheck` helpers in
`acctest_test.go`. They're gated on `resource.Test`'s built-in `TF_ACC=1`
requirement plus a `PreCheck` that skips unless credentials
(`GRAPHIANT_ACCESS_TOKEN`, or `GRAPHIANT_USERNAME`+`GRAPHIANT_PASSWORD`) are
set:

```bash
export TF_ACC=1
export GRAPHIANT_ACCESS_TOKEN="..."
go test ./internal/provider/... -run TestAcc -v
```

Without `TF_ACC=1` (e.g. plain `go test ./...` in CI's `test` job), every
`TestAcc*` skips immediately and touches nothing.

When adding a new resource or data source, add a matching acceptance test
following the existing pattern: a `TestAcc<Name>Resource` function using
`resource.Test`, with steps for create+read, `ImportState` (if the resource
implements `ResourceWithImportState` — use a composite id via
`ImportStateIdFunc` if `Read` needs more than the primary id, following
`alert_integration_resource_test.go`/`device_config_resource_test.go`), and
an in-place update. Acceptance tests create and delete real objects against a
live tenant — name them with a clearly identifiable random prefix (e.g. via
`acctest.RandomWithPrefix`, never a static string) so the same test can run
more than once concurrently (parallel CI runs, or a manual re-run overlapping
the nightly schedule) without colliding on names.

**Avoid hardcoded ids.** If a resource needs a foreign-key id (a LAN segment,
a site, a region, an alert rule, etc.), prefer creating that object in the
same test config and referencing its `.id` (see
`gateway_resource_test.go`/`public_vif_resource_test.go`'s throwaway
`graphiant_lan_segment`, or `alert_notification_resource_test.go`'s
`data.graphiant_alert_rules.all.rules[0].rule_id` lookup) over a hardcoded
number — this both removes any dependency on the test tenant's existing
data and makes the test fully parallel-safe. Only fall back to a hardcoded
placeholder when the referenced object genuinely can't be created or looked
up dynamically (a physical device, a prefix set/routing policy with no write
path, an enterprise identity) — in that case use `testAccPreCheckHardcoded`
instead of `testAccPreCheck`, so the test never runs automatically in CI
(`test.yml`'s `acceptance` job never sets `GRAPHIANT_ACC_HARDCODED_IDS`) and
only runs locally once a maintainer has edited the placeholder for their own
tenant and opted in via that env var. Document the placeholder in a comment
either way.

**Temporarily disabling a test.** If a test needs to stop running in CI for a
period (e.g. investigating a live-tenant failure) without deleting it, use
`testAccPreCheckHardcoded`'s sibling `testAccPreCheckDisabled` instead of
`testAccPreCheck`. Same shape: the test never runs automatically in CI
(`test.yml`'s `acceptance` job never sets `GRAPHIANT_ACC_RUN_DISABLED`) and
only runs locally once opted in via that env var. Prefer this over adding a
`-skip` flag to the `go test` invocation in `test.yml` — keeping the skip on
the test itself (with a comment explaining why) keeps the reason visible next
to the test instead of buried in CI config, and self-documents when it's safe
to remove.

### Sanity check (no Terraform required)

Before building the provider and setting up a dev override, `cmd/sanity`
gives a quick, Terraform-independent check that credentials and connectivity
work: it resolves auth exactly like the provider's own `Configure` does
(`GRAPHIANT_ACCESS_TOKEN`, or `GRAPHIANT_USERNAME`+`GRAPHIANT_PASSWORD`, and
optionally `GRAPHIANT_API_HOST`/`GRAPHIANT_HOST`), logs in, and lists the
current edge summary:

```bash
export GRAPHIANT_ACCESS_TOKEN="..."
make sanity
```

For a check that goes through the real provider binary and the real
Terraform plugin protocol instead of talking to the SDK directly,
`scripts/terraform-sanity.sh` builds the provider, wires up a throwaway
[dev override](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers)
(without touching your real `~/.terraformrc`), and runs
`scripts/terraform-sanity/main.tf` — the same `graphiant_edges` lookup, but
through `terraform apply`:

```bash
export GRAPHIANT_ACCESS_TOKEN="..."
make sanity-tf
```

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
- Keep resource files structurally consistent with their siblings (schema →
  model struct → `build*`/`apply*` conversion helpers → CRUD →
  `ImportState`) so the provider stays predictable to extend.
- Don't add attributes, resources, or data sources speculatively — only for
  API surface the provider actually needs to expose.
- Handle errors explicitly and surface them via `resp.Diagnostics`, never by
  panicking or silently dropping them. Use `apiErrorDetail(err)` (`errors.go`)
  rather than `err.Error()` for SDK call failures — `graphiant-sdk-go`'s
  `GenericOpenAPIError.Error()` only returns the HTTP status line (e.g.
  `"400 Bad Request"`); `apiErrorDetail` appends the actual response body.
- Close every SDK response body with `closeBody(httpResp)` (`util.go`), not
  `httpResp.Body.Close()` directly — the latter trips `golangci-lint`'s
  `errcheck` and there's no reason to inline the nil check at every call site.
- On a **resource** (not data source) schema, add
  `stringplanmodifier.UseStateForUnknown()` to `Computed`-only attributes
  that don't change as a side effect of updating other fields (most of them
  — e.g. `id`). Without it, the attribute shows `(known after apply)` on
  every single plan, not just when something relevant actually changed.
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
  (`go vet` + `go test -race`, no credentials needed), `acceptance`
  (`TestAcc*` against a live tenant if `GRAPHIANT_ACCESS_TOKEN` or
  `GRAPHIANT_USERNAME`+`GRAPHIANT_PASSWORD` are configured as repository
  secrets/variables — otherwise those tests self-skip and the job still
  passes), and `sanity` (`make sanity` + `make sanity-tf` against the same
  credentials — skipped, not failed, if none are configured). `acceptance`
  and `sanity` also run nightly via a `schedule` trigger. See
  [Repository secrets](#repository-secrets) below for what to configure.
- **[lint.yml](.github/workflows/lint.yml)** — `golangci-lint`, a `gofmt`
  check, `terraform fmt -check` over `examples/`, and the `docs` drift check
  described above.

### Repository secrets

None of these are required for `build`, `test`, or `lint.yml` to pass — only
to exercise `acceptance`/`sanity` against a live tenant, and to cut a
release. Add them under **Settings → Secrets and variables → Actions** in
GitHub.

| Name | Used by | Required? | Notes |
|------|---------|------------|-------|
| `GRAPHIANT_ACCESS_TOKEN` | `acceptance`, `sanity` | One of this or the username/password pair below | A bearer token; takes precedence over username/password. |
| `GRAPHIANT_USERNAME` | `acceptance`, `sanity` | — | Not secret-sensitive; can be added as a repository **variable** instead of a secret. |
| `GRAPHIANT_PASSWORD` | `acceptance`, `sanity` | — | Pair with `GRAPHIANT_USERNAME`. |
| `GRAPHIANT_HOST` | `acceptance`, `sanity` | No | Defaults to `https://api.graphiant.com`; only add if your test tenant uses a different host. Can be a repository variable. |
| `GPG_PRIVATE_KEY` | `release.yml` | Yes, to release | ASCII-armored private key whose public key is registered with the Terraform Registry's publisher settings for this provider. |
| `PASSPHRASE` | `release.yml` | Yes, to release | Passphrase for `GPG_PRIVATE_KEY`. |

`GITHUB_TOKEN` is provided automatically by GitHub Actions for both
workflows — never add it yourself. Without `GRAPHIANT_ACCESS_TOKEN` (or the
username/password pair), `acceptance` and `sanity` self-skip cleanly rather
than fail — see [Testing](#testing) above. Without `GPG_PRIVATE_KEY`/
`PASSPHRASE`, `release.yml` fails outright, since GoReleaser can't sign the
checksum file the Registry requires; see
[Releasing](#releasing) below for how to generate and register that key.

**Never point these at a production tenant.** `acceptance` creates and
deletes real objects; use a disposable test tenant.

## Releasing

This provider's version tracks the Graphiant platform/SDK release it was
built and tested against, rather than an independent SemVer sequence — e.g.
`v26.8.0` pairs with `graphiant-sdk-go v26.8.0`. When re-syncing against a
new SDK release, tag with the matching version; for a provider-only fix
against the same SDK version, increment the patch component instead (e.g.
`v26.8.1`).

**[release.yml](.github/workflows/release.yml)** builds cross-platform
binaries via [GoReleaser](https://goreleaser.com) (per
[`.goreleaser.yml`](.goreleaser.yml)), signs the checksum file with GPG
(required for the Terraform Registry), and publishes a GitHub release. It
triggers either way:

- **Push a tag matching `v*`** (e.g. `git tag v26.8.0 && git push origin
  v26.8.0`) — goes straight to the GoReleaser job, same as before this
  workflow had a dispatch option.
- **Run the workflow manually** from the Actions tab (`workflow_dispatch`,
  with a `version` input) — a `tag` job first checks that the caller has
  `admin` or `maintain` permission on the repo, then creates and pushes the
  tag (a no-op if it already exists) before GoReleaser runs. Use this when
  you'd rather not tag locally, or want repo permissions to gate who can cut
  a release.

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

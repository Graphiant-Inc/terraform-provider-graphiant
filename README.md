# Graphiant Terraform Provider

[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/dl/)
[![Terraform](https://img.shields.io/badge/terraform-1.0+-844FBA.svg)](https://developer.hashicorp.com/terraform/downloads)
[![Terraform Plugin Framework](https://img.shields.io/badge/terraform--plugin--framework-1.19-844FBA.svg)](https://developer.hashicorp.com/terraform/plugin/framework)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Documentation](https://img.shields.io/badge/docs-latest-brightgreen.svg)](https://docs.graphiant.com/docs/graphiant-terraform-provider)
[![Tests](https://github.com/Graphiant-Inc/terraform-provider-graphiant/actions/workflows/test.yml/badge.svg)](https://github.com/Graphiant-Inc/terraform-provider-graphiant/actions/workflows/test.yml)
[![Lint](https://github.com/Graphiant-Inc/terraform-provider-graphiant/actions/workflows/lint.yml/badge.svg)](https://github.com/Graphiant-Inc/terraform-provider-graphiant/actions/workflows/lint.yml)

Infrastructure as Code for [Graphiant Network-as-a-Service (NaaS)](https://www.graphiant.com),
built on [`terraform-plugin-framework`](https://developer.hashicorp.com/terraform/plugin/framework)
and backed directly by the [`graphiant-sdk-go`](https://github.com/Graphiant-Inc/graphiant-sdk-go)
client and models.

Refer to [Graphiant Docs](https://docs.graphiant.com) to get started with Graphiant NaaS offerings.

## Graphiant API Authentication

This provider authenticates to the Graphiant API using an **access token** or
**username/password** — the same credentials used across Graphiant's
[Ansible collection](https://github.com/Graphiant-Inc/graphiant-playbooks) and SDKs.

```bash
# Set the Graphiant API endpoint URL (optional — defaults to https://api.graphiant.com)
export GRAPHIANT_API_HOST="https://api.graphiant.com"
```

### Option 1: Set a Graphiant API access token

```bash
# Fetch a Graphiant API access token using the graphiant CLI
graphiant login
source ~/.graphiant/env.sh

# Verify the variable is set without printing its value
[ -n "$GRAPHIANT_ACCESS_TOKEN" ] && echo "GRAPHIANT_ACCESS_TOKEN is set"
```

The provider accepts `access_token` in the provider block and honors
**`GRAPHIANT_ACCESS_TOKEN`** when set — it takes precedence over
username/password (see [Authentication resolution](#authentication-resolution)
below).

### Option 2: Set Graphiant portal user login credentials

```bash
# Add to your shell profile (~/.zshrc, ~/.bashrc, etc.) or export for the session
export GRAPHIANT_USERNAME="your_username"
export GRAPHIANT_PASSWORD="your_password"

# Verify the variables are set without printing their values. Avoid echoing them
# or piping env through grep — that exposes secrets.
[ -n "$GRAPHIANT_USERNAME" ] && echo "GRAPHIANT_USERNAME is set"
[ -n "$GRAPHIANT_PASSWORD" ] && echo "GRAPHIANT_PASSWORD is set"
```

Pass `username`/`password` (or your secrets-manager equivalents) into the
provider block as needed.

### Authentication resolution

The provider resolves credentials in this order (see
[`internal/provider/provider.go`](internal/provider/provider.go) and
[`internal/provider/client.go`](internal/provider/client.go)):

1. `access_token` (provider block) — used as-is as a bearer token.
2. `username`/`password` (provider block) — exchanged for a token via the
   Graphiant login endpoint.
3. `GRAPHIANT_ACCESS_TOKEN`, or `GRAPHIANT_USERNAME` + `GRAPHIANT_PASSWORD`
   (environment) — same resolution, read from the environment via the SDK's
   own `graphiant_sdk.AuthorizationBearerFromEnvOrLogin`.

| Provider attribute | Environment variable                    | Sensitive |
|---------------------|------------------------------------------|-----------|
| `host`              | `GRAPHIANT_API_HOST` or `GRAPHIANT_HOST` | no        |
| `access_token`      | `GRAPHIANT_ACCESS_TOKEN`                 | yes       |
| `username`          | `GRAPHIANT_USERNAME`                     | no        |
| `password`          | `GRAPHIANT_PASSWORD`                     | yes       |

See [SECURITY.md](SECURITY.md) for credential-handling guidance.

## 📚 Documentation

- **Official documentation**: [Graphiant Docs](https://docs.graphiant.com) —
  product context for the NaaS offerings this provider manages.
- **Provider guide**: [Graphiant Terraform Provider](https://docs.graphiant.com/docs/graphiant-terraform-provider)
- **REST API reference**: [Graphiant Portal REST API](https://docs.graphiant.com/docs/graphiant-portal-rest-api)
- **Go SDK**: [graphiant-sdk-go](https://github.com/Graphiant-Inc/graphiant-sdk-go) —
  the client this provider is built on.
- **Terraform Plugin Framework**: [developer.hashicorp.com](https://developer.hashicorp.com/terraform/plugin/framework)
- **Terraform Registry**: [Graphiant Provider on the Terraform Registry](https://registry.terraform.io/providers/Graphiant-Inc/graphiant/latest) —
  install instructions, version history, and generated docs.
- **Provider reference docs**: generated under [`docs/`](docs/) via
  `tfplugindocs` — see [Full documentation](#full-documentation) below.
- **Changelog**: [CHANGELOG.md](CHANGELOG.md) — version history and release notes.
- **Security policy**: [SECURITY.md](SECURITY.md) — security best practices
  and vulnerability reporting.

## Features

### Key features

- **No codegen** — every resource/data source is hand-written directly
  against `graphiant-sdk-go` request/response structs, verified field by
  field rather than generated from a spec.
- **Composite import IDs where needed** — resources whose `Read` needs more
  than the primary id (e.g. `graphiant_device_config`,
  `graphiant_alert_integration`) implement a documented `"<a>:<b>"` import
  id instead of silently breaking `terraform import`.
- **Explicit scope boundaries** — every domain this provider doesn't cover
  (yet) is called out in the relevant resource's schema description, not
  silently dropped.
- **Full acceptance-test coverage** — every resource and data source has a
  `TestAcc*` test exercising create/read/update/import against a live
  tenant, gated on `TF_ACC=1` and credentials (see [Testing](#testing)).
- **Flexible auth** — a static bearer token or a username/password pair,
  configurable via the provider block or environment variables (see
  [Graphiant API Authentication](#graphiant-api-authentication) above).

### Resource & data source coverage

The SDK covers roughly a thousand Graphiant REST endpoints; this provider
currently exposes a core CRUD set, chosen for having well-defined
create/read/update/delete endpoints:

- **Sites** (`graphiant_site`) — name, notes, location, route tag.
- **IAM users** (`graphiant_user`) — the API has no general profile-update
  endpoint, so every configurable field forces recreation on change.
- **IAM groups** (`graphiant_group`) — permissions and member management.
- **Global app lists** (`graphiant_app_list`) — named groups of apps for policies.
- **Custom apps** (`graphiant_custom_app`) — URL/IP/port-based app matches.
- **Global content filters** (`graphiant_content_filter`) — domain-category
  blocking rules scoped to all sites or a site list.
- **Global site lists** (`graphiant_site_list`) — reusable groups of sites or route tags.
- **Enterprise/MSP** (`graphiant_enterprise`) — enterprise and MSP tenants,
  including credits/billing (`credit_limit`, `enterprise_contract`) — the API
  has no standalone credits resource.
- **Gateway service** (`graphiant_gateway`) — region/VRF-scoped gateways with
  core IPsec configuration.
- **Data exchange — public VIF** (`graphiant_public_vif`) — gateway Public VIF
  services.
- **Data exchange — local** (`graphiant_extranet`) — intra-tenant route-sharing
  policies between segments/sites.
- **Data exchange — partner** (`graphiant_b2b_producer_service`,
  `graphiant_b2b_customer`, `graphiant_b2b_match`, `graphiant_b2b_consumer`) —
  the B2B peer-to-peer/client-to-server workflow: producer service → customer
  invite → match → consumer accept.
- **Software rollouts** (`graphiant_software_rollout`) — upgrade rollout
  campaigns.
- **Device activation & decommission** (`graphiant_device_bringup`,
  `graphiant_device_decommission`) — action-shaped resources (see their
  descriptions for what "delete" does and doesn't do).
- **Data assurance** (`graphiant_assurance_global`,
  `graphiant_assurance_classified_application`) — SLA assurance config and
  custom app classification rules. ~20 time-windowed analytics report
  endpoints in this domain are intentionally not exposed (see below).
- **Alerts** (`graphiant_alert_integration`, `graphiant_alert_notification`) —
  delivery integrations (Zendesk/Slack webhook/PagerDuty/Opsgenie/Opsramp) and
  notification routing config.
- **Route tags** (`graphiant_route_tag`) — enterprise route tags; create+delete
  only, no update endpoint exists.
- **Device config** (`graphiant_device_config`) — pushes device configuration
  via the same generic endpoint (`V1DevicesDeviceIdConfigPut`) that
  graphiant-playbooks' Ansible modules use for BGP/interfaces/NAT policy/
  traffic policy/site-to-site VPN/etc. This resource only covers
  `maintenance_mode` and, on edge devices, the `*_enabled` toggles — the
  fields that round-trip cleanly on read. It does not cover those other
  config domains; see its schema description for why.
- **LAN segments** (`graphiant_lan_segment`) — global LAN segments;
  create+delete only, no update endpoint exists.
- **Devices (read-only)** — `graphiant_device` data source for a single
  onboarded edge device by ID. This provider does not manage device
  network configuration.
- **Edge/site monitoring (read-only)** — `graphiant_edges` (fleet summary),
  `graphiant_site_devices` (per-site device list with maintenance/VRRP
  state — the closest real proxy for "site health", which doesn't exist as
  a distinct endpoint).
- **Alerts (read-only)** — `graphiant_alert_records`, `graphiant_alert_rules`.
- **Troubleshooting/diagnostics (read-only)** — `graphiant_troubleshooting_device`,
  `graphiant_troubleshooting_site`. Diagnostic actions that are async
  (ping/traceroute/speedtest/packet-capture) or fire-and-forget (BGP reset,
  reboot, etc.) are not exposed — see below.
- **Routing (read-only)** — `graphiant_routing_policy`, `graphiant_prefix_set`
  (lookup by id — the API has no write path for either object at all).
- **Data assurance (read-only)** — `graphiant_assurance_flex_algos`,
  `graphiant_assurance_dnsproxy_entries` (the latter is read-only rather than
  a resource because its delete endpoint has no way to target a specific
  entry — see its schema description).
- **Reference lookups (read-only)** — `graphiant_domain_categories` (pairs
  with `content_filter`'s `rules[].domain_category_id`), `graphiant_regions`
  (pairs with `gateway`/`public_vif`/`device_config`'s region fields),
  `graphiant_ipsec_profiles`.

## Quick start

### Prerequisites

- [Go](https://go.dev/dl/) 1.25+ (only needed to build the provider yourself;
  see [Local development](#local-development))
- [Terraform](https://developer.hashicorp.com/terraform/downloads) 1.0+

### Example

```hcl
terraform {
  required_providers {
    graphiant = {
      source = "Graphiant-Inc/graphiant"
    }
  }
}

provider "graphiant" {
  # host, access_token, username, and password can also be set via
  # GRAPHIANT_API_HOST, GRAPHIANT_ACCESS_TOKEN, GRAPHIANT_USERNAME, and
  # GRAPHIANT_PASSWORD instead of being hardcoded here.
  host         = "https://api.graphiant.com"
  access_token = var.graphiant_access_token
}

resource "graphiant_site" "hq" {
  name  = "Headquarters"
  notes = "Managed by Terraform"

  location {
    address_line1 = "123 Main St"
    city          = "San Jose"
    state_code    = "CA"
    country_code  = "US"
  }
}

resource "graphiant_group" "network_admins" {
  name        = "network-admins"
  description = "Full network configuration access"

  permissions {
    network_configuration = "readWrite"
  }
}

resource "graphiant_user" "jane" {
  email      = "jane@example.com"
  first_name = "Jane"
  last_name  = "Doe"
  group_id   = graphiant_group.network_admins.id
}
```

> **Note:** to build against an unreleased change instead of the published
> version, use a [dev override](#local-development) pointing at a local build.

## Resources & data sources

| Type | Name | Notes |
|------|------|-------|
| Resource | `graphiant_site` | Create/update/delete sites |
| Resource | `graphiant_user` | Create/delete users; all configurable fields are force-new (no update endpoint) |
| Resource | `graphiant_group` | Create/update/delete IAM groups, permissions, and members |
| Resource | `graphiant_app_list` | Create/update/delete global app lists |
| Resource | `graphiant_content_filter` | Create/update/delete global content filters |
| Resource | `graphiant_custom_app` | Create/update/delete custom apps |
| Resource | `graphiant_site_list` | Create/delete global site lists; `name` is force-new (no update endpoint field) |
| Resource | `graphiant_enterprise` | Create/update/delete enterprise/MSP tenants, including credits/billing |
| Resource | `graphiant_gateway` | Create/update/delete gateways (core + single-peer IPsec) |
| Resource | `graphiant_public_vif` | Create/update/delete Public VIF data exchange services |
| Resource | `graphiant_extranet` | Create/update/delete local (intra-tenant) data exchange policies |
| Resource | `graphiant_b2b_producer_service` | Create/update/delete a B2B partner producer service |
| Resource | `graphiant_b2b_customer` | Create/update/delete a B2B partner customer invite |
| Resource | `graphiant_b2b_match` | Create/update/delete a B2B partner match (customer ↔ producer service) |
| Resource | `graphiant_b2b_consumer` | Create/update/delete a B2B partner consumer accept; not importable (see its description) |
| Resource | `graphiant_software_rollout` | Create/update/delete software upgrade rollout campaigns |
| Resource | `graphiant_device_bringup` | Action-shaped: trigger a bulk device bringup/activation status |
| Resource | `graphiant_device_decommission` | Action-shaped: drive the hardware-return decommission workflow |
| Resource | `graphiant_assurance_global` | Create/update/delete a global SLA assurance config |
| Resource | `graphiant_assurance_classified_application` | Create/update/delete a data-assurance app classification rule |
| Resource | `graphiant_alert_integration` | Create/update/delete an alert delivery integration |
| Resource | `graphiant_alert_notification` | Create/update/delete alert notification routing config; `rule_id_list` is force-new |
| Resource | `graphiant_route_tag` | Create/delete an enterprise route tag; no update endpoint exists |
| Resource | `graphiant_device_config` | Push device config (maintenance_mode + edge `*_enabled` toggles) via the generic device-config endpoint |
| Resource | `graphiant_lan_segment` | Create/delete a global LAN segment; no update endpoint exists |
| Data source | `graphiant_device` | Look up one onboarded device by `id` (read-only) |
| Data source | `graphiant_edges` | Current edge device summary list, optionally filtered |
| Data source | `graphiant_site_devices` | Per-site device list with maintenance/VRRP state |
| Data source | `graphiant_alert_records` | Current top-level alert records |
| Data source | `graphiant_alert_rules` | The fixed alert rule catalog with enabled state |
| Data source | `graphiant_assurance_flex_algos` | Flex algo reference list |
| Data source | `graphiant_assurance_dnsproxy_entries` | Current DNS proxy filter entries (read-only — see its description) |
| Data source | `graphiant_troubleshooting_device` | Device health snapshot |
| Data source | `graphiant_troubleshooting_site` | Site status snapshot with per-edge state |
| Data source | `graphiant_routing_policy` | Global routing policies, looked up by `ids` |
| Data source | `graphiant_prefix_set` | Global prefix sets, looked up by `ids` |
| Data source | `graphiant_domain_categories` | Content-filter domain category catalog |
| Data source | `graphiant_regions` | Graphiant region catalog |
| Data source | `graphiant_ipsec_profiles` | Global IPsec profiles and their reference counts |

## Examples

Every resource and data source has a runnable example under
[`examples/`](examples/), laid out the way `terraform-plugin-docs` (and the
Terraform Registry) expect:

```
examples/
├── provider/provider.tf
├── resources/graphiant_site/{resource.tf,import.sh}
├── resources/graphiant_user/{resource.tf,import.sh}
├── resources/graphiant_group/{resource.tf,import.sh}
├── resources/graphiant_app_list/{resource.tf,import.sh}
├── resources/graphiant_content_filter/{resource.tf,import.sh}
├── resources/graphiant_custom_app/{resource.tf,import.sh}
├── resources/graphiant_site_list/{resource.tf,import.sh}
├── resources/graphiant_enterprise/{resource.tf,import.sh}
├── resources/graphiant_gateway/{resource.tf,import.sh}
├── resources/graphiant_public_vif/{resource.tf,import.sh}
├── resources/graphiant_extranet/{resource.tf,import.sh}
├── resources/graphiant_b2b_producer_service/{resource.tf,import.sh}
├── resources/graphiant_b2b_customer/{resource.tf,import.sh}
├── resources/graphiant_b2b_match/{resource.tf,import.sh}
├── resources/graphiant_b2b_consumer/resource.tf
├── resources/graphiant_software_rollout/{resource.tf,import.sh}
├── resources/graphiant_device_bringup/resource.tf
├── resources/graphiant_device_decommission/resource.tf
├── resources/graphiant_assurance_global/{resource.tf,import.sh}
├── resources/graphiant_assurance_classified_application/{resource.tf,import.sh}
├── resources/graphiant_alert_integration/{resource.tf,import.sh}
├── resources/graphiant_alert_notification/{resource.tf,import.sh}
├── resources/graphiant_route_tag/{resource.tf,import.sh}
├── resources/graphiant_device_config/{resource.tf,import.sh}
├── resources/graphiant_lan_segment/{resource.tf,import.sh}
├── data-sources/graphiant_device/data-source.tf
├── data-sources/graphiant_edges/data-source.tf
├── data-sources/graphiant_site_devices/data-source.tf
├── data-sources/graphiant_alert_records/data-source.tf
├── data-sources/graphiant_alert_rules/data-source.tf
├── data-sources/graphiant_assurance_flex_algos/data-source.tf
├── data-sources/graphiant_assurance_dnsproxy_entries/data-source.tf
├── data-sources/graphiant_troubleshooting_device/data-source.tf
├── data-sources/graphiant_troubleshooting_site/data-source.tf
├── data-sources/graphiant_routing_policy/data-source.tf
├── data-sources/graphiant_prefix_set/data-source.tf
├── data-sources/graphiant_domain_categories/data-source.tf
├── data-sources/graphiant_regions/data-source.tf
└── data-sources/graphiant_ipsec_profiles/data-source.tf
```

These are the same snippets `tfplugindocs` embeds in generated docs and on
the [Terraform Registry page](https://registry.terraform.io/providers/Graphiant-Inc/graphiant/latest) —
copy one directly into a `.tf` file to get started, or adapt the
[Quick start](#quick-start) example above.

## Full documentation

Per-attribute reference docs are generated from the resource/data-source
`Description` strings in `internal/provider/*.go` and the examples in
`examples/`, via [`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs):

```bash
make docs
```

This writes `docs/index.md`, `docs/resources/*.md`, and `docs/data-sources/*.md`
— don't hand-edit those files. CI (`docs` job in
[lint.yml](.github/workflows/lint.yml)) fails a PR if `docs/` is out of sync.

## Testing

- **Unit tests** (`go test ./...`, no credentials needed) — validate every
  resource/data source's schema (`TestResourceSchemas`/
  `TestDataSourceSchemas`).
- **Acceptance tests** (`TestAcc*` in `internal/provider/*_test.go`) exercise
  every resource's create/read/update/import cycle (where importable) and
  every data source, against a live Graphiant tenant. See
  [How-to: run the acceptance tests](#how-to-run-the-acceptance-tests) below.
- **Sanity checks** (no `.tf` config or CI wiring needed) — two lightweight,
  local-only smoke tests confirm credentials/connectivity and the provider
  binary work before you reach for the full acceptance suite. See
  [How-to: run the sanity checks](#how-to-run-the-sanity-checks) below.

## How-tos

### How-to: import an existing resource

Most resources support `terraform import`:

```bash
terraform import graphiant_site.hq 12345
```

A few resources need more than the primary id to read themselves back, and
use a composite `"<a>:<b>"` import id instead — check the resource's schema
description (`terraform providers schema` or the generated docs) if a plain
numeric id doesn't work:

```bash
# graphiant_device_config: "<device_id>:<device_type>"
terraform import graphiant_device_config.edge1 12345:edge

# graphiant_alert_integration: "<enterprise_id>:<integration_id>"
terraform import graphiant_alert_integration.pagerduty 1:98765
```

`graphiant_b2b_consumer`, `graphiant_device_bringup`, and
`graphiant_device_decommission` aren't importable at all — see their schema
descriptions for why (no get-by-id endpoint, or the resource is action-shaped
rather than a real object).

### How-to: run the sanity checks

Before reaching for the full acceptance suite, two lightweight, local-only
checks confirm credentials/connectivity and the provider binary work.
Neither needs `TF_ACC=1`, and neither writes or deletes anything in your
tenant — both only read the edge summary.

**`make sanity`** — talks to the SDK directly (`cmd/sanity`), with no
Terraform involved at all. Fastest way to confirm your credentials and
network access to the Graphiant API work:

```bash
export GRAPHIANT_ACCESS_TOKEN="..."   # or GRAPHIANT_USERNAME + GRAPHIANT_PASSWORD
export GRAPHIANT_API_HOST="https://api.graphiant.com"   # optional, this is the default

make sanity
```

It resolves auth exactly like the provider's own `Configure` does, logs in,
and prints a table of the current edge summary (device ID, hostname, status,
role, site). Fails fast with a clear message if no credentials are set.

**`make sanity-tf`** — goes through the real provider binary and the real
Terraform plugin protocol instead (`scripts/terraform-sanity.sh`), confirming
the compiled provider itself behaves correctly end to end, not just that
credentials/connectivity work:

```bash
export GRAPHIANT_ACCESS_TOKEN="..."   # or GRAPHIANT_USERNAME + GRAPHIANT_PASSWORD

make sanity-tf
```

Requires `terraform` on your `PATH`. It builds the provider binary, wires up
a **throwaway** CLI config with a
[dev override](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers)
pointing at that binary (never touches your real `~/.terraformrc`), then runs
`terraform apply` against `scripts/terraform-sanity/main.tf` (a
`graphiant_edges` data source plus outputs) — skipping `terraform init`,
since dev overrides make it unnecessary. All `.terraform/`, lock file, and
state artifacts it creates are removed automatically on exit, even on
failure.

### How-to: run the acceptance tests

Acceptance tests are gated on `TF_ACC=1` and credentials — without both, they
skip cleanly (`go test ./...` alone never talks to a real tenant):

```bash
export TF_ACC=1
export GRAPHIANT_ACCESS_TOKEN="..."   # or GRAPHIANT_USERNAME + GRAPHIANT_PASSWORD
export GRAPHIANT_API_HOST="https://api.graphiant.com"

go test ./internal/provider/... -run TestAccSiteResource -v
```

Never run these against a production tenant: they create and delete real
objects. Most tests are fully self-contained (random names, and any needed
foreign-key id — a LAN segment, a site, a region — is created or looked up
within the test itself), so they're also safe to run more than once in
parallel. A handful reference an object this provider has no way to create
on demand (a physical device, a prefix set/routing policy, an enterprise
identity) — these use `testAccPreCheckHardcoded` instead of the standard
`PreCheck` and never run in CI. To run one locally: edit the hardcoded
placeholder in that test file to a real id from your own test tenant, then
set `GRAPHIANT_ACC_HARDCODED_IDS=1` in addition to the credentials above.

### How-to: look up reference/platform ids before writing config

Several resources take an id that comes from a fixed, platform-defined list
rather than something you create — look it up with the matching data source
instead of guessing a number:

```hcl
data "graphiant_regions" "all" {}

resource "graphiant_gateway" "primary" {
  region_id = [for r in data.graphiant_regions.all.regions : r.id if r.name == "us-west"][0]
  # ...
}
```

The same pattern applies to `graphiant_domain_categories` (for
`graphiant_content_filter`'s `rules[].domain_category_id`) and
`graphiant_ipsec_profiles` (for `graphiant_gateway`'s
`ipsec_gateway.vpn_profile`).

### How-to: wire up a B2B ("partner") data exchange

The B2B resources form a 4-step chain — producer service, then a customer
invite, then a match linking the two, then the customer's consumer accept —
each referencing the previous step's `id`:

```hcl
resource "graphiant_b2b_producer_service" "svc" {
  service_name = "partner-peering-service"
  service_type = "peering_service"
  policy = {
    service_lan_segment = 100
  }
}

resource "graphiant_b2b_customer" "partner" {
  name = "Partner Co"
  type = "non-graphiant"
  invite = {
    admin_emails = ["admin@partner.example"]
  }
}

resource "graphiant_b2b_match" "match" {
  customer_id = graphiant_b2b_customer.partner.id
  match = {
    service_id  = graphiant_b2b_producer_service.svc.id
    lan_segment = 100
  }
}

resource "graphiant_b2b_consumer" "accept" {
  customer_id = graphiant_b2b_customer.partner.id
  match_id    = graphiant_b2b_match.match.id
  service_id  = graphiant_b2b_producer_service.svc.id
  policy = {
    consumer_lan_segments = {
      "200" = { consumer_prefixes = ["10.50.0.0/16"] }
    }
  }
}
```

`graphiant_b2b_consumer` isn't importable, so once applied, manage it only
through this same Terraform config going forward.

### How-to: push device config safely

`graphiant_device_config` writes through the same generic endpoint the
Graphiant Ansible collection uses for many config domains, but this resource
only exposes the handful of fields that reliably round-trip on read
(`maintenance_mode`, `region`, edge `*_enabled` toggles, `description`,
`local_web_server_password`). Create/update push the change and then poll the
resulting job until it completes or fails:

```hcl
resource "graphiant_device_config" "edge1" {
  device_id        = 12345
  device_type      = "edge"
  maintenance_mode = true
}
```

It does not cover BGP, interfaces, NAT/security/traffic policy, site-to-site
VPN, LAG, DHCP relay, NTP, OSPFv2, static routes, VRRP, MACsec, or prefix/port
lists — see the resource's schema description for the full list and why.

## Project Structure

```
terraform-provider-graphiant/
├── main.go                      # Provider entrypoint (providerserver.Serve)
├── cmd/sanity/                  # Terraform-independent smoke test (make sanity)
├── scripts/terraform-sanity/    # Terraform-level smoke test (make sanity-tf)
├── internal/provider/           # Provider schema, resources, data sources — hand-written, no codegen
│   ├── provider.go               #   schema, auth resolution, Resources()/DataSources() registries
│   ├── client.go                 #   API client + token resolution helpers
│   ├── *_resource.go              #   one file per managed resource
│   ├── *_data_source.go           #   one file per read-only data source
│   └── *_test.go                  #   unit tests + TestAcc* acceptance tests
├── examples/                    # Runnable .tf config per resource/data source (tfplugindocs layout)
├── docs/                        # Generated attribute reference (tfplugindocs) — do not hand-edit
├── .github/workflows/            # CI: build/test/acceptance, lint, release (GoReleaser + GPG)
├── CHANGELOG.md
├── CONTRIBUTING.md
├── SECURITY.md
├── CODE_OF_CONDUCT.md
└── LICENSE
```

## Local development

```bash
git clone https://github.com/Graphiant-Inc/terraform-provider-graphiant.git
cd terraform-provider-graphiant
make build
```

Point Terraform at your local build instead of the registry with a
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

With the override in place, skip `terraform init` and run `terraform plan`/
`apply` directly — Terraform will use your local binary. See
[CONTRIBUTING.md](CONTRIBUTING.md) for the full development workflow,
project layout, and testing guidance.

## 🤝 Contributing

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md) for the
development workflow, project layout, and pull request checklist. Please
also review our [Code of Conduct](CODE_OF_CONDUCT.md).

## 📄 License

Licensed under the [MIT License](LICENSE).

## 🆘 Support

- **Official documentation**: [Graphiant Docs](https://docs.graphiant.com)
- **Changelog**: [CHANGELOG.md](CHANGELOG.md) — version history and release notes
- **Security**: [SECURITY.md](SECURITY.md) — security policy and vulnerability reporting
- **Issues & feature requests**: [GitHub Issues](https://github.com/Graphiant-Inc/terraform-provider-graphiant/issues)
- **Email**: support@graphiant.com

## 🔗 Related Projects

- [Graphiant SDK Go](https://github.com/Graphiant-Inc/graphiant-sdk-go) — the client this provider is built on
- [Graphiant SDK Python](https://github.com/Graphiant-Inc/graphiant-sdk-python)
- [Graphiant Playbooks](https://github.com/Graphiant-Inc/graphiant-playbooks) — Ansible collection and Terraform modules for cloud gateway connectivity

---

**Made with ❤️ by the Graphiant Team**

# Graphiant Terraform Provider

[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Terraform Plugin Framework](https://img.shields.io/badge/terraform--plugin--framework-1.19-844FBA.svg)](https://developer.hashicorp.com/terraform/plugin/framework)
[![Tests](https://github.com/Graphiant-Inc/terraform-provider-graphiant/actions/workflows/test.yml/badge.svg)](https://github.com/Graphiant-Inc/terraform-provider-graphiant/actions/workflows/test.yml)
[![Lint](https://github.com/Graphiant-Inc/terraform-provider-graphiant/actions/workflows/lint.yml/badge.svg)](https://github.com/Graphiant-Inc/terraform-provider-graphiant/actions/workflows/lint.yml)
[![Docs](https://img.shields.io/badge/docs-docs%2F-844FBA.svg)](docs/index.md)

A Terraform provider for [Graphiant Network-as-a-Service (NaaS)](https://www.graphiant.com),
built on [`terraform-plugin-framework`](https://developer.hashicorp.com/terraform/plugin/framework)
and backed by the [`graphiant-sdk-go`](https://github.com/Graphiant-Inc/graphiant-sdk-go) client.

More product context: [Graphiant Docs](https://docs.graphiant.com).

> **Status:** pre-release. This provider has not yet been published to the
> Terraform Registry — see [Local development](#local-development) to build
> and use it against a real Graphiant tenant in the meantime.

## Table of contents

| Section | Contents |
|---------|----------|
| [Documentation & links](#documentation--links) | Official guides, API reference, SDK |
| [Features](#features) | What the provider manages |
| [Requirements](#requirements) | Go and Terraform versions |
| [Quick start](#quick-start) | Provider configuration and a first resource |
| [Authentication](#authentication) | Access token vs. username/password |
| [Resources & data sources](#resources--data-sources) | Full inventory |
| [Examples](#examples) | Ready-to-run `.tf` config per resource/data source |
| [Full documentation](#full-documentation) | Generated attribute reference (`docs/`) |
| [Testing](#testing) | Unit tests vs. acceptance tests against a live tenant |
| [Local development](#local-development) | Build and use a local provider binary |
| [Security](#security) | Credential handling |
| [Contributing](#contributing) | PR workflow |
| [License](#license) | MIT |
| [Version history](#version-history) | Changelog |
| [Support](#support) | Links and contact |

## Documentation & links

| Resource | Link |
|----------|------|
| **Go SDK** | [graphiant-sdk-go](https://github.com/Graphiant-Inc/graphiant-sdk-go) |
| **REST API** | [Graphiant Portal REST API](https://docs.graphiant.com/docs/graphiant-portal-rest-api) |
| **Terraform Plugin Framework** | [developer.hashicorp.com](https://developer.hashicorp.com/terraform/plugin/framework) |
| **Changelog** | [CHANGELOG.md](CHANGELOG.md) |

## Features

- **Sites** — create, update, and delete Graphiant sites (name, notes,
  location), plus `graphiant_site`/`graphiant_sites` data sources.
- **IAM groups** — manage groups and their per-area permissions
  (`graphiant_group`), plus `graphiant_group`/`graphiant_groups` data sources.
- **IAM users** — manage users and their group assignment (`graphiant_user`),
  plus `graphiant_user`/`graphiant_users` data sources.
- **Devices (read-only)** — `graphiant_device`/`graphiant_devices` data
  sources for onboarded edge devices. This provider does not manage device
  network configuration.
- **Flexible auth** — a static bearer token or a username/password pair,
  configurable via the provider block or environment variables.

## Requirements

- [Go](https://go.dev/dl/) 1.25+ (only needed to build the provider yourself;
  see [Local development](#local-development))
- [Terraform](https://developer.hashicorp.com/terraform/downloads) 1.0+

## Quick start

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

  location = {
    address_line1 = "123 Main St"
    city          = "San Jose"
    state_code    = "CA"
    country_code  = "US"
  }
}

resource "graphiant_group" "network_admins" {
  name        = "network-admins"
  description = "Full network configuration access"

  permissions = {
    network_configuration = "write"
    monitoring_and_troubleshooting = "write"
  }
}

resource "graphiant_user" "jane" {
  email      = "jane@example.com"
  first_name = "Jane"
  last_name  = "Doe"
  group_id   = graphiant_group.network_admins.id
}

data "graphiant_devices" "all" {}

output "device_count" {
  value = length(data.graphiant_devices.all.devices)
}
```

> **Note:** until this provider is published to the Terraform Registry,
> `terraform init` cannot download it from `source = "Graphiant-Inc/graphiant"`
> — you'll need a [dev override](#local-development) pointing at a local
> build.

## Authentication

The provider resolves credentials in this order (see
[`internal/provider/client.go`](internal/provider/client.go)):

1. `access_token` (provider block) or `GRAPHIANT_ACCESS_TOKEN` (env) — used
   as-is as a bearer token.
2. `username`/`password` (provider block) or `GRAPHIANT_USERNAME` /
   `GRAPHIANT_PASSWORD` (env) — exchanged for a token via the Graphiant login
   endpoint.

| Provider attribute     | Environment variable                        | Sensitive |
|-------------------------|----------------------------------------------|-----------|
| `host`                  | `GRAPHIANT_API_HOST` or `GRAPHIANT_HOST`     | no        |
| `access_token`          | `GRAPHIANT_ACCESS_TOKEN`                     | yes       |
| `username`              | `GRAPHIANT_USERNAME`                         | no        |
| `password`              | `GRAPHIANT_PASSWORD`                         | yes       |
| `insecure_skip_verify`  | —                                             | no        |

`insecure_skip_verify` disables TLS certificate validation and should only be
used against a trusted lab/on-prem controller — never in production. See
[SECURITY.md](SECURITY.md) for more on handling these values safely.

## Resources & data sources

| Type | Name | Notes |
|------|------|-------|
| Resource | `graphiant_site` | Create/update/delete sites |
| Resource | `graphiant_group` | Create/update/delete IAM groups and their permissions |
| Resource | `graphiant_user` | Create/update/delete IAM users |
| Data source | `graphiant_site` | Look up one site by `id` |
| Data source | `graphiant_sites` | List all sites |
| Data source | `graphiant_group` | Look up one IAM group by `id` |
| Data source | `graphiant_groups` | List all IAM groups |
| Data source | `graphiant_user` | Look up one user by `id` (email) |
| Data source | `graphiant_users` | List all users |
| Data source | `graphiant_device` | Look up one onboarded device by `id` (read-only) |
| Data source | `graphiant_devices` | List all onboarded devices (read-only) |

Group membership is managed by setting `group_id` on `graphiant_user`; there
is no separate group-membership resource yet.

## Examples

Every resource and data source has a runnable example under
[`examples/`](examples/), laid out the way `terraform-plugin-docs` (and the
Terraform Registry) expect:

```
examples/
├── provider/provider.tf                       # provider configuration
├── resources/graphiant_site/
│   ├── resource.tf                             # minimal working config
│   └── import.sh                               # terraform import example
├── resources/graphiant_group/...
├── resources/graphiant_user/...
└── data-sources/graphiant_site/data-source.tf  # one per data source
    data-sources/graphiant_sites/...
    data-sources/graphiant_group/...
    data-sources/graphiant_groups/...
    data-sources/graphiant_user/...
    data-sources/graphiant_users/...
    data-sources/graphiant_device/...
    data-sources/graphiant_devices/...
```

These are the same snippets embedded in [`docs/`](docs/) and on the Terraform
Registry page once published — copy one directly into a `.tf` file to get
started, or adapt the [Quick start](#quick-start) example above.

## Full documentation

Generated, per-attribute reference docs live in [`docs/`](docs/index.md):

| Doc | Covers |
|-----|--------|
| [`docs/index.md`](docs/index.md) | Provider configuration schema |
| [`docs/resources/`](docs/resources/) | `graphiant_site`, `graphiant_group`, `graphiant_user` — full schema + import syntax |
| [`docs/data-sources/`](docs/data-sources/) | All eight data sources — full schema |

These are generated from the resource/data-source `Description` strings in
`internal/provider/*.go` and the examples in `examples/`, via
[`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs) — **don't
hand-edit files under `docs/`**. After changing a schema description or an
example, regenerate with:

```bash
make docs
```

CI (`docs` job in [lint.yml](.github/workflows/lint.yml)) fails a PR if
`docs/` is out of sync with the schema and examples.

## Testing

This provider has two layers of tests, both under `internal/provider/`:

- **Unit tests** (`go test ./...`, no credentials needed) — validate every
  resource/data source's schema (`TestResourceSchemas`/
  `TestDataSourceSchemas`) and any non-trivial `expand`/`flatten` conversion
  logic. These run on every PR (`test` job in
  [test.yml](.github/workflows/test.yml)).
- **Acceptance tests** (`TestAcc*`, e.g. `TestAccSiteResource`) — exercise a
  full create → read → update → import → delete cycle for each resource
  against a **live Graphiant tenant**, using
  [`terraform-plugin-testing`](https://developer.hashicorp.com/terraform/plugin/testing).
  They're gated on `TF_ACC=1` (the upstream convention — without it,
  `resource.Test` self-skips) *and* on Graphiant credentials being present
  (`GRAPHIANT_ACCESS_TOKEN`, or `GRAPHIANT_USERNAME` + `GRAPHIANT_PASSWORD` —
  the same variables the provider itself reads). Run them locally with:

  ```bash
  export GRAPHIANT_ACCESS_TOKEN="..."   # or GRAPHIANT_USERNAME + GRAPHIANT_PASSWORD
  make testacc
  ```

  In CI, the `acceptance` job in [test.yml](.github/workflows/test.yml) runs
  on every PR/push and nightly (`schedule` trigger), reading the same
  variables from repository secrets/variables. **Acceptance tests create and
  delete real sites/groups/users** (all named with an `acctest.RandomWithPrefix`
  prefix like `tf-acc-site-...` so they're easy to spot and clean up if a run
  is interrupted) — point `GRAPHIANT_API_HOST`/`GRAPHIANT_HOST` at a
  non-production tenant if you have one. If no credentials are configured
  (e.g. a fork's PR), the job still runs but every `TestAcc*` self-skips with
  an explanation, so CI stays green.

See [CONTRIBUTING.md](CONTRIBUTING.md#testing) for how to add tests for a new
resource or data source.

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

## Security

See [SECURITY.md](SECURITY.md) for supported versions, how to report a
vulnerability, and credential-handling guidance specific to this provider
(sensitive attributes, state file exposure, `insecure_skip_verify`, etc.).

## Contributing

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md) for the
development workflow, project layout, and pull request checklist. Please
also review our [Code of Conduct](CODE_OF_CONDUCT.md).

## License

Licensed under the [MIT License](LICENSE).

## Version history

See [CHANGELOG.md](CHANGELOG.md).

## Support

- **Issues & feature requests**: [GitHub Issues](https://github.com/Graphiant-Inc/terraform-provider-graphiant/issues)
- **Product docs**: [docs.graphiant.com](https://docs.graphiant.com)
- **Go SDK**: [graphiant-sdk-go](https://github.com/Graphiant-Inc/graphiant-sdk-go)

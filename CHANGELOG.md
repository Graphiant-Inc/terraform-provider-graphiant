# Changelog

All notable changes to the Graphiant Terraform Provider are documented here,
following the entry conventions from HashiCorp's
[Versioning and Changelog best practices](https://developer.hashicorp.com/terraform/plugin/best-practices/versioning):
entries are grouped as BREAKING CHANGES / NOTES / FEATURES / IMPROVEMENTS /
BUG FIXES, each prefixed with the affected subsystem. This project has not
yet made a tagged release; versioning will follow
[Semantic Versioning](https://semver.org/) once it does.

## (Unreleased)

NOTES:

* provider: Initial development release. No compatibility guarantees apply
  until a `v1.0.0` (or first tagged) release is cut.

FEATURES:

* provider: Initial provider scaffolding using `terraform-plugin-framework`,
  backed by [`graphiant-sdk-go`](https://github.com/Graphiant-Inc/graphiant-sdk-go).
  Configuration: `host`, `access_token`, `username`, `password`,
  `insecure_skip_verify`, with `GRAPHIANT_API_HOST` / `GRAPHIANT_HOST` /
  `GRAPHIANT_ACCESS_TOKEN` / `GRAPHIANT_USERNAME` / `GRAPHIANT_PASSWORD`
  environment variable fallbacks.
* resource/graphiant_site: New resource for managing sites (name, notes,
  location).
* resource/graphiant_group: New resource for managing IAM groups (name,
  description, permissions, time-window restrictions).
* resource/graphiant_user: New resource for managing IAM users (email, name,
  group assignment, time zone).
* data-source/graphiant_site: New data source to look up a single site by id.
* data-source/graphiant_sites: New data source to list all sites.
* data-source/graphiant_group: New data source to look up a single IAM group
  by id.
* data-source/graphiant_groups: New data source to list all IAM groups.
* data-source/graphiant_user: New data source to look up a single user by id.
* data-source/graphiant_users: New data source to list all users.
* data-source/graphiant_device: New read-only data source to look up a single
  onboarded device by id.
* data-source/graphiant_devices: New read-only data source to list all
  onboarded devices.

IMPROVEMENTS:

* resource: Set `UseStateForUnknown` on computed attributes whose value isn't
  a deterministic side effect of the resource's own `Update` (e.g. a site's
  `tags`/`edge_count`, a user's `verified`/`phone_number`). Previously these
  showed as `(known after apply)` on every plan, even when nothing relevant
  had changed. Attributes that genuinely change on every update (e.g. a
  site's `updated_at`) intentionally keep the old behavior.
* provider: API errors now surface the response body instead of just the
  HTTP status line: `graphiant-sdk-go`'s `GenericOpenAPIError.Error()` only
  returns something like `"400 Bad Request"`, so a new `apiErrorDetail`
  helper (`util.go`) appends the actual response body to every diagnostic.
* provider: Added `tflog` debug/trace logging throughout resource CRUD and
  data source `Read` methods for `TF_LOG`-based troubleshooting.
* resource/graphiant_user: Added validation that `email` must look like an
  email address.
* resource: `graphiant_group`/`graphiant_site`/`graphiant_user` now reject
  empty required strings.
* resource/graphiant_group: `time_window_start` and `time_window_end` must
  now be set together.
* testing: Added `terraform-plugin-testing`-based acceptance tests
  (`TestAccSiteResource`, `TestAccGroupResource`, `TestAccUserResource`,
  `TestAccDevicesDataSource`) covering full create/read/update/import cycles
  against a live Graphiant tenant, gated on `TF_ACC=1` and Graphiant
  credentials (`internal/provider/acctest_test.go` and `*_test.go`).
* docs: Added an `examples/` directory with a runnable `.tf` (and
  `import.sh` where applicable) for the provider block, every resource, and
  every data source, in the layout `terraform-plugin-docs`/the Terraform
  Registry expect.
* docs: Added generated attribute-reference docs under `docs/` via
  [`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs)
  (`make docs`), sourced from schema `Description`s and `examples/`.
* ci: Added an `acceptance` job to `test.yml` (credential-gated, plus a
  nightly `schedule` trigger) running the new acceptance tests, and a `docs`
  job in `lint.yml` failing PRs where `docs/` drifts from schema/examples,
  alongside a `terraform fmt -check` job over `examples/`.
* meta: Added CI/CD setup (GitHub Actions workflows for tests, lint, and
  tagged releases via GoReleaser + GPG-signed checksums for the Terraform
  Registry) and project documentation (README, CONTRIBUTING, SECURITY,
  CODE_OF_CONDUCT, LICENSE (MIT)).

BUG FIXES:

* provider: Removed 22 unchecked `httpRes.Body.Close()` errors (now a single
  `closeBody` helper) and three dead helper functions
  (`locationAttrTypes`, `int32PtrFromInt64`, `stringSlicePtr`), both flagged
  by `golangci-lint`.

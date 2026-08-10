# Changelog

All notable changes to the Graphiant Terraform Provider will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
This project has not yet made a tagged release; versioning will follow
[Semantic Versioning](https://semver.org/) once it does.

## [Unreleased]

### Added

- Initial provider scaffolding using `terraform-plugin-framework`, backed by
  [`graphiant-sdk-go`](https://github.com/Graphiant-Inc/graphiant-sdk-go).
- Provider configuration: `host`, `access_token`, `username`, `password`,
  `insecure_skip_verify`, with `GRAPHIANT_API_HOST` / `GRAPHIANT_HOST` /
  `GRAPHIANT_ACCESS_TOKEN` / `GRAPHIANT_USERNAME` / `GRAPHIANT_PASSWORD`
  environment variable fallbacks.
- **Resources:**
  - `graphiant_site` — manage sites (name, notes, location).
  - `graphiant_group` — manage IAM groups (name, description, permissions,
    time-window restrictions).
  - `graphiant_user` — manage IAM users (email, name, group assignment, time
    zone).
- **Data sources:**
  - `graphiant_site` / `graphiant_sites`
  - `graphiant_group` / `graphiant_groups`
  - `graphiant_user` / `graphiant_users`
  - `graphiant_device` / `graphiant_devices` (read-only device summaries)
- Project documentation: README, CONTRIBUTING, SECURITY, CODE_OF_CONDUCT,
  LICENSE (MIT).
- CI/CD setup: GitHub Actions workflows for tests, lint, and tagged releases
  (GoReleaser + GPG-signed checksums for the Terraform Registry), plus
  `.gitignore`, `.golangci.yml`, `.goreleaser.yml`, and
  `terraform-registry-manifest.json`.

### Changed

- Resource schemas now set `UseStateForUnknown` on computed attributes whose
  value isn't a deterministic side effect of the resource's own `Update`
  (e.g. a site's `tags`/`edge_count`, a user's `verified`/`phone_number`).
  Previously these showed as `(known after apply)` on every plan, even when
  nothing relevant had changed. Attributes that genuinely change on every
  update (e.g. a site's `updated_at`) intentionally keep the old behavior.
- API errors now surface the response body instead of just the HTTP status
  line: `graphiant-sdk-go`'s `GenericOpenAPIError.Error()` only returns
  something like `"400 Bad Request"`, so a new `apiErrorDetail` helper
  (`util.go`) appends the actual response body to every diagnostic.
- Added `tflog` debug/trace logging throughout resource CRUD and data source
  `Read` methods for `TF_LOG`-based troubleshooting.
- Added schema-level validation: `graphiant_user.email` must look like an
  email address; `graphiant_group`/`graphiant_site`/`graphiant_user` reject
  empty required strings; `graphiant_group.time_window_start` and
  `time_window_end` must be set together.

### Fixed

- Removed 22 unchecked `httpRes.Body.Close()` errors (now a single
  `closeBody` helper) and three dead helper functions
  (`locationAttrTypes`, `int32PtrFromInt64`, `stringSlicePtr`), both flagged
  by `golangci-lint`.

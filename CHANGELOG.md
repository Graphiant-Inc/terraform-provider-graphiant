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

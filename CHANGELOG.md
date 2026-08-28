# Changelog

All notable changes to the Graphiant Terraform Provider are documented here,
following the entry conventions from HashiCorp's
[Versioning and Changelog best practices](https://developer.hashicorp.com/terraform/plugin/best-practices/versioning):
entries are grouped as BREAKING CHANGES / NOTES / FEATURES / IMPROVEMENTS /
BUG FIXES, each prefixed with the affected subsystem. Versioning tracks the
Graphiant platform/SDK release the provider was built and tested against
(e.g. `26.8.0` targets `graphiant-sdk-go` `v26.8.0`), not an independent
SemVer sequence starting from `0.1.0`/`1.0.0` — see
[CONTRIBUTING.md](CONTRIBUTING.md#releasing) for how that plays out when
cutting a new release.

## 26.8.0

NOTES:

* provider: First tagged release, targeting `graphiant-sdk-go` `v26.8.0`.
  As a new provider, expect the API surface below to keep growing across
  releases; see the scoping note below for what's covered so far.
* provider: This provider intentionally covers a subset of the Graphiant API
  — resources are only added for endpoints that support full create/read/
  update/delete lifecycle management (declarative infrastructure), not for
  one-shot actions, analytics/telemetry queries, or session/account
  management. Several domains (device BGP/interfaces/NAT/security/traffic
  policy, global batch config, most action-style endpoints) are deliberately
  out of scope for now. See CONTRIBUTING.md for the full API-surface audit
  behind this scoping, and each resource/data source's schema description for
  what it does and does not cover.

FEATURES:

* provider: Provider scaffolding using `terraform-plugin-framework`, backed
  directly by [`graphiant-sdk-go`](https://github.com/Graphiant-Inc/graphiant-sdk-go)
  models (no intermediate codegen step). Configuration: `host`,
  `access_token`, `username`, `password`, `insecure_skip_verify`, with
  `GRAPHIANT_API_HOST` / `GRAPHIANT_HOST` / `GRAPHIANT_ACCESS_TOKEN` /
  `GRAPHIANT_USERNAME` / `GRAPHIANT_PASSWORD` environment variable fallbacks.

  Core:

  * resource/graphiant_site: Enterprise sites (name, notes, location).
  * resource/graphiant_user: Users (email, name, group assignment, time
    zone).
  * resource/graphiant_group: IAM permissions groups (name, description,
    permissions, time-window restrictions).
  * resource/graphiant_enterprise: Enterprise/MSP tenants, including
    credits/billing fields (`enterprise_contract`).
  * data-source/graphiant_device: Looks up a single onboarded device by id.
  * data-source/graphiant_edges: Current edge device summary/status list.
  * data-source/graphiant_site_devices: Per-site device list with
    maintenance/VRRP/staging state.

  Policy scoping and reusable object lists:

  * resource/graphiant_site_list: Global site lists (member sites/route
    tags); `name` is force-new (no rename endpoint).
  * resource/graphiant_route_tag: Enterprise route tags — a 3-level
    hierarchical tag used to scope policies to sites/segments.
  * resource/graphiant_content_filter: Global content filters (domain-category
    blocking rules, scoped to all sites or a site list).
  * resource/graphiant_app_list: Named lists of apps (custom or built-in),
    used as a match target in policies.
  * resource/graphiant_custom_app: Custom application definitions, matched
    by URL, IP lists/prefixes, and/or port ranges.
  * resource/graphiant_lan_segment: Global LAN segments; no update endpoint,
    every field is force-new.
  * data-source/graphiant_domain_categories: The platform-defined domain
    category catalog used by `graphiant_content_filter`'s
    `rules[].domain_category_id`.
  * data-source/graphiant_regions: Graphiant regions, used by
    `graphiant_gateway`/`graphiant_public_vif`'s `region_id`.
  * data-source/graphiant_prefix_set: Global prefix sets, looked up by id
    (read-only — no create/update/delete endpoint exists for this object).
  * data-source/graphiant_routing_policy: Global routing policies, looked up
    by id (read-only, same reason as `graphiant_prefix_set`).

  Gateway, data exchange, and B2B ("partner") services:

  * resource/graphiant_gateway: Graphiant gateway services.
  * resource/graphiant_public_vif: Gateway Public VIF data exchange
    services.
  * resource/graphiant_extranet: Local (intra-tenant) data exchange
    policies, sharing routes between segments/sites.
  * resource/graphiant_b2b_producer_service: The producer side of a B2B
    ("partner") data exchange — part of a 4-object workflow (producer
    service → customer invite → match → consumer accept).
  * resource/graphiant_b2b_customer: B2B customer invites.
  * resource/graphiant_b2b_match: B2B matches, linking a customer to a
    producer service.
  * resource/graphiant_b2b_consumer: The consumer side of a B2B match; not
    importable (no get-by-consumer-id endpoint exists).
  * data-source/graphiant_ipsec_profiles: Global IPsec profiles and how many
    gateway configs reference each one, used by `graphiant_gateway`'s
    `ipsec_gateway.vpn_profile`.

  Device lifecycle and configuration:

  * resource/graphiant_device_bringup: Triggers a bulk device
    bringup/activation status transition.
  * resource/graphiant_device_decommission: Drives the hardware-return
    decommission workflow (request → approve → clear).
  * resource/graphiant_software_rollout: Software upgrade rollout campaigns
    for a set of devices.
  * resource/graphiant_device_config: Pushes device configuration via the
    generic device-config endpoint, covering `maintenance_mode`, `region`,
    and edge feature-enable scalars, plus description/local web server
    password. Asynchronous: create/update poll the returned job until it
    completes. Does not cover BGP/interfaces/NAT/security/traffic policy,
    site-to-site VPN, LAG, DHCP relay, NTP, OSPFv2, static routes, VRRP,
    MACsec, or prefix/port lists — see its schema description for why.

  Data assurance, alerting, and troubleshooting:

  * resource/graphiant_assurance_global: Global SLA assurance configurations
    (which apps/LANs/sites to monitor and which flex algo to score against).
  * resource/graphiant_assurance_classified_application: Custom application
    classification rules for data assurance.
  * resource/graphiant_alert_integration: Alert delivery integrations
    (Zendesk/Slack webhook/PagerDuty/Opsgenie/Opsramp).
  * resource/graphiant_alert_notification: Alert notification routing
    config, binding rules to delivery channels/recipients.
  * data-source/graphiant_alert_records: Current top-level (parent) alert
    records.
  * data-source/graphiant_alert_rules: The fixed catalog of alert rules and
    their current enabled state.
  * data-source/graphiant_assurance_flex_algos: The platform-defined flex
    algo reference list used by `graphiant_assurance_global`'s `flex_algo`.
  * data-source/graphiant_assurance_dnsproxy_entries: Current DNS proxy
    filter entries for data assurance (read-only).
  * data-source/graphiant_troubleshooting_device: A device health snapshot.
  * data-source/graphiant_troubleshooting_site: A site status snapshot,
    including per-edge device status.

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
* resource: Resources whose `Read` needs more than the primary id
  (`graphiant_device_config` needs a device type, `graphiant_alert_integration`
  needs the owning enterprise id) implement a composite `"<a>:<b>"` import id
  via a custom `ImportState`, documented in each resource's schema
  description, rather than a plain id passthrough that would silently break
  `terraform import`.
* testing: Added `terraform-plugin-testing`-based acceptance tests covering
  every resource (create/read/update/import where supported) and every data
  source, gated on `TF_ACC=1` and Graphiant credentials
  (`internal/provider/acctest_test.go` and `*_test.go`). Tests requiring
  tenant-specific ids (device/site/region ids, etc.) use documented
  placeholder values that need adjusting per test tenant.
* docs: Added an `examples/` directory with a runnable `.tf` (and
  `import.sh` where applicable) for the provider block, every resource, and
  every data source, in the layout `terraform-plugin-docs`/the Terraform
  Registry expect.
* docs: Added generated attribute-reference docs under `docs/` via
  [`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs)
  (`make docs`), sourced from schema `Description`s and `examples/`.
* ci: Added an `acceptance` job to `test.yml` (credential-gated, plus a
  nightly `schedule` trigger) running the acceptance test suite, and a
  `docs` job in `lint.yml` failing PRs where `docs/` drifts from
  schema/examples, alongside a `terraform fmt -check` job over `examples/`.
* meta: Added CI/CD setup (GitHub Actions workflows for tests, lint, and
  tagged releases via GoReleaser + GPG-signed checksums for the Terraform
  Registry) and project documentation (README, CONTRIBUTING, SECURITY,
  CODE_OF_CONDUCT, LICENSE (MIT)).

BUG FIXES:

* provider: Removed 22 unchecked `httpRes.Body.Close()` errors (now a single
  `closeBody` helper) and three dead helper functions
  (`locationAttrTypes`, `int32PtrFromInt64`, `stringSlicePtr`), both flagged
  by `golangci-lint`.

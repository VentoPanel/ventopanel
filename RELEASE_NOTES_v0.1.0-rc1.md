# VentoPanel v0.1.0-rc1

First release candidate for the VentoPanel MVP foundation.

## Highlights

- Production-oriented SSL workflow with asynchronous issue/renew tasks, scheduler, and observability metrics.
- Redis-based distributed locks and Asynq task deduplication to improve idempotency for deploy/provision/SSL flows.
- Audit trail with filterable API, cursor pagination, and optional `total_count`.
- Team-based ACL enforcement for site and server endpoints with role-aware authorization (`owner/admin/viewer`).
- Monitoring bundle with Prometheus, alert rules, and Grafana dashboard provisioning.
- CI pipeline with OpenAPI lint, Go lint, unit tests, and Postgres-backed integration tests.

## API and Contract

- Added OpenAPI spec for `GET /api/v1/audit/status-events` in `openapi/audit-status-events.yaml`.
- Cursor contract:
  - `before=<RFC3339Nano timestamp>,<event_id>`
  - `next_cursor` in response
  - optional `include_total=true` (returns `total_count`, ignoring cursor pagination)

## Testing and Quality

- Added integration coverage for:
  - audit endpoint filters, cursor behavior, and `include_total`
  - server connect transition auditing
  - ACL allowed/forbidden scenarios for site and server routes
- CI now includes:
  - OpenAPI lint
  - `make lint`
  - `make test`
  - `make test-integration`

## Notes

- This is a release candidate tag intended for validation and smoke testing before final `v0.1.0`.

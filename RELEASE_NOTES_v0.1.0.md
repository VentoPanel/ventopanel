# VentoPanel v0.1.0

Initial public MVP release of VentoPanel.

## Highlights

- Production-oriented SSL workflow with asynchronous issue/renew tasks, scheduler, and observability metrics.
- Redis-based distributed locks and Asynq task deduplication to improve idempotency for deploy/provision/SSL flows.
- Audit trail with filterable API, cursor pagination, and optional `total_count`.
- Team-based ACL enforcement for site and server endpoints with role-aware authorization (`owner/admin/viewer`).
- Monitoring bundle with Prometheus, alert rules, and Grafana dashboard provisioning.
- CI pipeline with OpenAPI lint, Go lint, unit tests, and Postgres-backed integration tests.

## API and Contract

- OpenAPI spec for `GET /api/v1/audit/status-events` in `openapi/audit-status-events.yaml`.
- Cursor contract:
  - `before=<RFC3339Nano timestamp>,<event_id>`
  - `next_cursor` in response
  - optional `include_total=true` (returns `total_count`, ignoring cursor pagination)

## Testing and Quality

- Integration coverage includes:
  - audit endpoint filters, cursor behavior, and `include_total`
  - server connect transition auditing
  - ACL allowed/forbidden scenarios for site and server routes
- CI includes:
  - OpenAPI lint
  - `make lint`
  - `make test`
  - `make test-integration`

## Notes

- This release establishes the MVP baseline for iterative delivery and production hardening.

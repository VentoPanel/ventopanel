# Changelog

All notable changes to this project are documented in this file.

## v0.1.7 - 2026-04-28

### Added

- Frontend scaffolding under `frontend/` using Next.js 15 App Router, TypeScript, Tailwind CSS 3, Shadcn UI primitives, and TanStack Query v5.
- Dashboard page with stat cards (server/site counts), servers table, and sites table.
- Servers and Sites list pages with status badges (`success`, `warning`, `destructive`).
- Login page: JWT token input stored in `localStorage`, redirects to dashboard.
- Route guard in dashboard layout: unauthenticated users are redirected to `/login`.
- API client in `frontend/lib/api.ts`: reads token from `localStorage`, adds `Authorization` header, redirects on 401/403.
- `frontend/Dockerfile` with multi-stage build (`node:20-alpine`) and standalone Next.js output.
- `frontend` service added to `deployments/docker/docker-compose.yaml` and `docker-compose.override.yaml` (port 3000, log rotation, restart policy).

## v0.1.6 - 2026-04-28

### Added

- Added `make smoke-prod-auth SITE_ID=<id>` to automate production auth smoke checks: health endpoint, ACL deny request, ACL deny metrics, and `status_events` audit verification.
- Documented the new Makefile shortcut in `OPERATIONS.md` for repeatable post-release verification on servers.

## v0.1.5 - 2026-04-28

### Added

- Added `deployments/docker/docker-compose.override.yaml` with production-safe defaults (`restart: unless-stopped`) and Docker json-file log rotation for `api`, `postgres`, and `redis`.
- Expanded `OPERATIONS.md` with merged compose commands (`base + override`) and a post-release ACL deny smoke procedure covering health, metrics, and audit verification.

## v0.1.4 - 2026-04-28

### Added

- ACL deny observability with Prometheus metric `ventopanel_acl_denied_total` labeled by resource and deny reason.
- ACL deny audit events for server/site authorization failures, written to `status_events` with `to_status=access_denied`.
- Integration test coverage to verify deny decisions are persisted in audit storage.

## v0.1.0 - 2026-04-28

### Added

- Production-oriented SSL workflow with async issue/renew tasks, scheduler, and observability metrics.
- Redis distributed locks and Asynq task deduplication for deploy/provision/SSL idempotency.
- Audit trail subsystem with status transition events, filterable read API, and cursor pagination.
- OpenAPI spec for `GET /api/v1/audit/status-events` in `openapi/audit-status-events.yaml`.
- Monitoring bundle with Prometheus, alert rules, and Grafana dashboard provisioning.
- Integration tests for audit API and server connect audit transitions.
- CI workflow with OpenAPI lint, Go lint, unit tests, and Postgres-backed integration tests.

### Changed

- Refactored backend toward clean architecture boundaries (`domain/service/repository/transport`).
- Hardened server credential handling with AES-GCM encryption at rest.
- Improved server and site lifecycle safety via explicit transition state machine validation.
- Expanded HTTP API surface for server/site CRUD, connect/provision/deploy/SSL, metrics, and observability.
- Updated docs and Makefile commands for monitoring, integration testing, and CI usage.

### Notes

- `total_count` in audit listing is optional (`include_total=true`) and intentionally ignores cursor pagination.

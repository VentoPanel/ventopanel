# Changelog

All notable changes to this project are documented in this file.

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

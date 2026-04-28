# VentoPanel v0.1.3

Operations and deployment consistency patch release.

## Highlights

- Added `OPERATIONS.md` with a practical runbook for day-2 operations:
  - service lifecycle
  - health/smoke checks
  - incident log triage
  - Postgres backup/restore procedures
  - production config baseline
- Updated Docker Compose runtime behavior to rely on env-driven API configuration and removed obsolete compose schema version declaration.

## Why this matters

- Reduces production drift by ensuring server-side `.env` values are applied consistently.
- Improves operational readiness with explicit backup/restore and incident handling commands.

## Recommended post-upgrade checks

- `docker compose -f deployments/docker/docker-compose.yaml up -d --build`
- `curl http://127.0.0.1:8080/api/v1/health`
- Verify production security posture:
  - no-JWT protected endpoint -> `403`
  - `/api/v1/dev/token` in production -> `404`

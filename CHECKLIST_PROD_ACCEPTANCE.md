# Production Acceptance Checklist

Use this checklist before each production go-live.

## 1) Runtime and Deployment

- [ ] `docker compose -f deployments/docker/docker-compose.yaml up -d --build` completed successfully.
- [ ] `docker compose -f deployments/docker/docker-compose.yaml ps` shows `api`, `postgres`, `redis` healthy/running.
- [ ] API health endpoint returns OK: `GET /api/v1/health`.

## 2) Security Baseline

- [ ] `APP_ENV=production`.
- [ ] `AUTH_ALLOW_HEADER_FALLBACK=false`.
- [ ] `AUTH_JWT_SECRET` is strong and rotated as required.
- [ ] `AUTH_JWT_ISSUER` and `AUTH_JWT_AUDIENCE` are configured and validated by issued tokens.
- [ ] `POST /api/v1/dev/token` returns `404` in production.

## 3) Access Control and API Behavior

- [ ] Protected endpoint without JWT returns `403`.
- [ ] ACL by-id checks are correct (`200/403` as expected).
- [ ] ACL list filtering works (`GET /api/v1/sites` and `GET /api/v1/servers` only return accessible resources).

## 4) Data and Background Processing

- [ ] Migrations are applied (`servers`, `sites`, `teams`, `team_site_access`, `status_events` exist).
- [ ] Redis and worker processing are healthy (no recurring task failures in logs).
- [ ] Audit endpoint responds correctly (`/api/v1/audit/status-events`, cursor and `include_total` behavior).

## 5) Observability and Recovery

- [ ] Metrics endpoint available (`GET /metrics`).
- [ ] Log tail check completed (`docker compose ... logs --tail=200`).
- [ ] Fresh Postgres backup was created and restore procedure is validated (or recently drilled).

## 6) Final Gate

- [ ] Current commit/tag is documented in release notes.
- [ ] Rollback command and previous stable tag are known before traffic cutover.

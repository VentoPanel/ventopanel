# VentoPanel

The Go-powered Control Panel for Scalable Apps.

## Backend stack

- Go 1.26+
- Gin
- PostgreSQL 17
- Redis
- Asynq
- Clean Architecture

## Quick start

```bash
cp .env.example .env
make deps
make compose-up
make run
```

In development mode, you can mint a local JWT for testing:

```bash
curl -X POST "http://localhost:8080/api/v1/dev/token" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"dev-user","team_id":"dev-team","ttl_seconds":3600}'
```

## CI

- Workflow file: `.github/workflows/ci.yml`
- Runs on each push/PR:
  - OpenAPI lint for `openapi/audit-status-events.yaml`
  - `make lint`
  - `make test`
  - `make test-integration` (with Postgres 17 service and `TEST_POSTGRES_DSN`)

Important security variable:

- `APP_ENCRYPTION_KEY` must be 32 bytes. It is used to encrypt SSH passwords at rest.
- `AUTH_JWT_SECRET` secures JWT parsing in HTTP middleware. Change default value in non-dev environments.
- Production recommendation: set `AUTH_ALLOW_HEADER_FALLBACK=false` to require JWT-derived identity and disable header trust fallback.

## Monitoring (Prometheus + Grafana)

- Prometheus config: `deployments/monitoring/prometheus.yml`
- Alert rules: `deployments/monitoring/alert.rules.yml`
- Grafana dashboard JSON: `deployments/monitoring/grafana-dashboard-ssl.json`
- One-command stack: `deployments/monitoring/docker-compose.monitoring.yml`

Run monitoring:

```bash
docker compose -f deployments/monitoring/docker-compose.monitoring.yml up -d
```

Grafana:

- URL: `http://localhost:3000`
- User: `admin`
- Password: `admin`

The API exposes:

- `GET /metrics` - Prometheus metrics endpoint
- `GET /api/v1/observability/ssl` - internal JSON observability endpoint

## Audit API (cursor pagination)

Audit status events endpoint:

- `GET /api/v1/audit/status-events`
- OpenAPI spec: `openapi/audit-status-events.yaml`

Supported query params:

- `resource_type`, `resource_id`, `from`, `to`
- `since` (RFC3339)
- `before` cursor in format `<RFC3339Nano timestamp>,<event_id>`
- `limit` (1..500, default `100`)
- `include_total` (`true|false`, default `false`)

Notes:

- Results are ordered by `created_at DESC, id DESC`.
- Response includes `next_cursor` for the next page.
- `total_count` is returned only when `include_total=true`.
- `total_count` is calculated for current filters and ignores `before`.

Example request:

```bash
curl "http://localhost:8080/api/v1/audit/status-events?resource_type=site&limit=2&include_total=true"
```

Cursor pagination request (next page):

```bash
curl "http://localhost:8080/api/v1/audit/status-events?resource_type=site&limit=2&before=2026-04-28T09:00:00.123456789Z,d6f4f3d4-4b42-4e8d-a22a-6ef0d682ef4c"
```

Filter by specific resource and time window:

```bash
curl "http://localhost:8080/api/v1/audit/status-events?resource_type=server&resource_id=srv_001&since=2026-04-01T00:00:00Z&limit=50"
```

Run audit integration tests:

```bash
TEST_POSTGRES_DSN=postgres://vento:vento@localhost:5432/ventopanel?sslmode=disable make test-integration
```

Current integration coverage:

- `GET /api/v1/audit/status-events`: filters, `include_total`, cursor pagination via `before`, invalid `include_total`
- `POST /api/v1/servers/:id/connect`: success audit event, failure audit event, invalid transition (no status change/no audit), not found (`404`)

Example response:

```json
{
  "items": [
    {
      "id": "d6f4f3d4-4b42-4e8d-a22a-6ef0d682ef4c",
      "resource_type": "site",
      "resource_id": "site_123",
      "from_status": "deploying",
      "to_status": "deployed",
      "reason": "deployment completed",
      "task_id": "deploy:site_123",
      "created_at": "2026-04-28T09:00:00.123456789Z"
    }
  ],
  "next_cursor": "2026-04-28T09:00:00.123456789Z,d6f4f3d4-4b42-4e8d-a22a-6ef0d682ef4c",
  "total_count": 57
}
```

## Project layout

- `cmd/api` - application entrypoint
- `internal/service` - business logic
- `internal/repository` - data access
- `internal/transport/http` - HTTP delivery layer
- `internal/infra` - external integrations
- `internal/worker` - async jobs
- `migrations` - SQL migrations

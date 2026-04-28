# VentoPanel Operations Runbook

Minimal day-2 operations guide for single-host Docker deployments.

## 1) Service Lifecycle

Start/update stack:

```bash
cd /root/ventopanel
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  up -d --build
```

Stop stack:

```bash
cd /root/ventopanel
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  down
```

Restart API only:

```bash
cd /root/ventopanel
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  restart api
```

## 2) Health and Smoke Checks

Container status:

```bash
cd /root/ventopanel
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  ps
```

API health:

```bash
curl -sS http://127.0.0.1:8080/api/v1/health | jq .
```

Metrics endpoint:

```bash
curl -sS http://127.0.0.1:8080/metrics | head
```

ACL deny smoke (v0.1.4+):

```bash
# 1) create JWT with team_id that has no grants
TOKEN="<prod-jwt>"

# 2) should be 403 (deny path)
curl -sS -o /tmp/acl-deny.json -w "ACL_DENY:%{http_code}\n" \
  -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8080/api/v1/sites/<site_id_without_grant>

# 3) metric should include acl denies
curl -sS http://127.0.0.1:8080/metrics | grep ventopanel_acl_denied_total

# 4) audit should contain access_denied rows
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  exec -T postgres \
  psql -U vento -d ventopanel -c \
  "SELECT resource_type, resource_id, to_status, reason, created_at
   FROM status_events
   WHERE to_status='access_denied'
   ORDER BY created_at DESC
   LIMIT 10;"
```

Security sanity in production mode:

```bash
# should be 403
curl -sS -o /tmp/nojwt.json -w "NOJWT:%{http_code}\n" http://127.0.0.1:8080/api/v1/sites

# should be 404 when APP_ENV=production
curl -sS -o /tmp/devtoken.json -w "DEVTOKEN:%{http_code}\n" \
  -X POST "http://127.0.0.1:8080/api/v1/dev/token" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u1","team_id":"t1","ttl_seconds":3600}'
```

## 3) Logs and Incident Triage

Tail API logs:

```bash
cd /root/ventopanel
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  logs -f api
```

Last 200 lines all services:

```bash
cd /root/ventopanel
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  logs --tail=200
```

Useful checks:

- DB connectivity failures -> verify `POSTGRES_DSN` and postgres health.
- Redis task issues -> verify `REDIS_ADDR` and redis container status.
- Auth failures -> verify `AUTH_JWT_SECRET`, issuer/audience, and fallback mode.

## 4) Postgres Backup

Create backup:

```bash
mkdir -p /root/backups
docker exec -i ventopanel-postgres pg_dump -U vento -d ventopanel > /root/backups/ventopanel_$(date +%F_%H-%M-%S).sql
ls -lh /root/backups
```

Recommended:

- Keep daily backups and copy to off-host storage.
- Periodically validate restore with a drill.

## 5) Postgres Restore

Restore from SQL dump (destructive for current DB contents):

```bash
BACKUP_FILE=/root/backups/<file>.sql

docker exec -i ventopanel-postgres psql -U vento -d postgres -c "DROP DATABASE IF EXISTS ventopanel;"
docker exec -i ventopanel-postgres psql -U vento -d postgres -c "CREATE DATABASE ventopanel;"
docker exec -i ventopanel-postgres psql -U vento -d ventopanel < "$BACKUP_FILE"
```

After restore:

```bash
cd /root/ventopanel
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  restart api
curl -sS http://127.0.0.1:8080/api/v1/health | jq .
```

## 6) Config Baseline (Production)

Critical `.env` keys:

- `APP_ENV=production`
- `AUTH_ALLOW_HEADER_FALLBACK=false`
- `AUTH_JWT_SECRET=<strong random secret>`
- `AUTH_JWT_ISSUER=<issuer>`
- `AUTH_JWT_AUDIENCE=<audience>`
- `APP_ENCRYPTION_KEY=<exactly 32 bytes>`

When changing `.env`, rebuild/restart:

```bash
cd /root/ventopanel
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  up -d --build
```

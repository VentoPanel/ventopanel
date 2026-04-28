APP_NAME := ventopanel
CMD_DIR := ./cmd/api
BIN_DIR := ./bin
BINARY := $(BIN_DIR)/$(APP_NAME)

.PHONY: help deps run build test test-integration lint fmt clean compose-up compose-down monitoring-up monitoring-down monitoring-logs migrate-up migrate-down smoke-prod-auth

help:
	@echo "Available commands:"
	@echo "  make deps         - download and tidy dependencies"
	@echo "  make run          - run API locally"
	@echo "  make build        - build application binary"
	@echo "  make test         - run all tests"
	@echo "  make test-integration - run integration tests (requires TEST_POSTGRES_DSN)"
	@echo "  make lint         - run go vet"
	@echo "  make fmt          - format Go code"
	@echo "  make clean        - remove build artifacts"
	@echo "  make compose-up   - start local docker services"
	@echo "  make compose-down - stop local docker services"
	@echo "  make monitoring-up   - start Prometheus and Grafana"
	@echo "  make monitoring-down - stop monitoring stack"
	@echo "  make monitoring-logs - tail monitoring stack logs"
	@echo "  make migrate-up   - apply SQL migrations"
	@echo "  make migrate-down - rollback one SQL migration"
	@echo "  make smoke-prod-auth SITE_ID=<id> - run production ACL deny smoke"

deps:
	go mod download
	go mod tidy

run:
	go run $(CMD_DIR)

build:
	mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w" -o $(BINARY) $(CMD_DIR)

test:
	go test ./... -race -v

test-integration:
	@if [ -z "$$TEST_POSTGRES_DSN" ]; then echo "TEST_POSTGRES_DSN is required. Example:"; echo "TEST_POSTGRES_DSN=postgres://vento:vento@localhost:5432/ventopanel?sslmode=disable make test-integration"; exit 1; fi
	go test ./internal/transport/http -run "AuditStatusEvents|ServerConnect" -count=1 -v

lint:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -rf $(BIN_DIR)

compose-up:
	docker compose -f deployments/docker/docker-compose.yaml up -d --build

compose-down:
	docker compose -f deployments/docker/docker-compose.yaml down -v

monitoring-up:
	docker compose -f deployments/monitoring/docker-compose.monitoring.yml up -d

monitoring-down:
	docker compose -f deployments/monitoring/docker-compose.monitoring.yml down -v

monitoring-logs:
	docker compose -f deployments/monitoring/docker-compose.monitoring.yml logs -f

migrate-up:
	migrate -path ./migrations -database "postgres://vento:vento@localhost:5432/ventopanel?sslmode=disable" up

migrate-down:
	migrate -path ./migrations -database "postgres://vento:vento@localhost:5432/ventopanel?sslmode=disable" down 1

smoke-prod-auth:
	@if [ -z "$$SITE_ID" ]; then echo "SITE_ID is required. Example:"; echo "make smoke-prod-auth SITE_ID=935778e2-57fb-44d5-a85d-e49a46b7085f"; exit 1; fi
	@TOKEN=$$(python3 - <<'PY' \
import json, base64, hmac, hashlib, time, pathlib; \
def b64url(data): \
    return base64.urlsafe_b64encode(data).rstrip(b'=').decode(); \
env = {}; \
for line in pathlib.Path('.env').read_text(encoding='utf-8').splitlines(): \
    line = line.strip(); \
    if not line or line.startswith('#') or '=' not in line: \
        continue; \
    k, v = line.split('=', 1); \
    env[k.strip()] = v.strip(); \
secret = env['AUTH_JWT_SECRET']; \
iss = env.get('AUTH_JWT_ISSUER', ''); \
aud = env.get('AUTH_JWT_AUDIENCE', ''); \
now = int(time.time()); \
header = {'alg': 'HS256', 'typ': 'JWT'}; \
payload = {'team_id': '11111111-1111-1111-1111-111111111111', 'iss': iss, 'aud': aud, 'iat': now, 'nbf': now, 'exp': now + 3600}; \
h = b64url(json.dumps(header, separators=(',', ':')).encode()); \
p = b64url(json.dumps(payload, separators=(',', ':')).encode()); \
msg = f'{h}.{p}'.encode(); \
sig = b64url(hmac.new(secret.encode(), msg, hashlib.sha256).digest()); \
print(f'{h}.{p}.{sig}'); \
PY \
	); \
	echo "Health:"; curl -fsS http://127.0.0.1:8080/api/v1/health; echo; \
	echo "ACL deny request (expected 403):"; \
	curl -sS -o /tmp/acl-deny.json -w "ACL_DENY:%{http_code}\n" -H "Authorization: Bearer $$TOKEN" "http://127.0.0.1:8080/api/v1/sites/$$SITE_ID"; \
	cat /tmp/acl-deny.json; echo; \
	echo "ACL deny metric:"; curl -fsS http://127.0.0.1:8080/metrics | grep ventopanel_acl_denied_total; \
	echo "Latest access_denied events:"; \
	docker compose -f deployments/docker/docker-compose.yaml -f deployments/docker/docker-compose.override.yaml exec -T postgres psql -U vento -d ventopanel -c "SELECT resource_type, resource_id, to_status, reason, created_at FROM status_events WHERE to_status='access_denied' ORDER BY created_at DESC LIMIT 10;"

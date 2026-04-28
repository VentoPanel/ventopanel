APP_NAME := ventopanel
CMD_DIR := ./cmd/api
BIN_DIR := ./bin
BINARY := $(BIN_DIR)/$(APP_NAME)

.PHONY: help deps run build test test-integration lint fmt clean compose-up compose-down monitoring-up monitoring-down monitoring-logs migrate-up migrate-down

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

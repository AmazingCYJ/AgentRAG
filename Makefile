GOCACHE ?= $(CURDIR)/.cache/go-build
POSTGRES_COMPOSE ?= resources/docker/postgres.compose.yaml
POSTGRES_DSN ?= postgres://postgres:postgres@127.0.0.1:5432/ragent?sslmode=disable

.PHONY: run-api run-mcp test postgres-up postgres-down postgres-reset postgres-logs test-postgres

run-api:
	go run ./cmd/api

run-mcp:
	go run ./cmd/mcp-server

test:
	GOCACHE=$(GOCACHE) go test ./...

postgres-up:
	docker compose -f $(POSTGRES_COMPOSE) up -d --wait

postgres-down:
	docker compose -f $(POSTGRES_COMPOSE) down

postgres-reset:
	docker compose -f $(POSTGRES_COMPOSE) down -v
	docker compose -f $(POSTGRES_COMPOSE) up -d --wait

postgres-logs:
	docker compose -f $(POSTGRES_COMPOSE) logs -f postgres

test-postgres:
	AGENTRAG_POSTGRES_DSN="$(POSTGRES_DSN)" GOCACHE=$(GOCACHE) go test ./internal/platform/db -run TestPostgresRepositoryBootstrapSmoke -count=1 -v

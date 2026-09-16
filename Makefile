SHELL := /bin/sh
ENV_FILE := .env

.PHONY: env check-env db-up db-down migrate migrate-report migrate-002 api import embed backend-test

env:
	@if test -e "$(ENV_FILE)"; then \
		echo "$(ENV_FILE) already exists; left unchanged"; \
	else \
		install -m 600 .env.example "$(ENV_FILE)"; \
		echo "Created $(ENV_FILE); set POSTGRES_PASSWORD before use"; \
	fi

check-env:
	@test -f "$(ENV_FILE)" || { echo "Missing $(ENV_FILE); run 'make env'" >&2; exit 1; }
	@set -a; . ./$(ENV_FILE); set +a; \
		for name in POSTGRES_DB POSTGRES_USER POSTGRES_PASSWORD DATABASE_URL; do \
			eval "value=\$$$${name}"; \
			test -n "$$value" || { echo "$$name must be set in $(ENV_FILE)" >&2; exit 1; }; \
		done

db-up: check-env
	@docker compose --env-file "$(ENV_FILE)" up -d db

db-down: check-env
	@docker compose --env-file "$(ENV_FILE)" down

migrate-002: check-env
	@set -a; . ./$(ENV_FILE); set +a; docker compose --env-file "$(ENV_FILE)" exec -T db psql -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -f /docker-entrypoint-initdb.d/002_semantic_search.sql

migrate: check-env
	@set -a; . ./$(ENV_FILE); set +a; DATABASE_URL="$${ADMIN_DATABASE_URL:-$$DATABASE_URL}"; export DATABASE_URL; cd apps/backend && exec go run ./cmd/migrate -dir migrations

migrate-report: check-env
	@set -a; . ./$(ENV_FILE); set +a; DATABASE_URL="$${ADMIN_DATABASE_URL:-$$DATABASE_URL}"; export DATABASE_URL; cd apps/backend && exec go run ./cmd/migrate -report

api: check-env
	@set -a; . ./$(ENV_FILE); set +a; cd apps/backend && exec go run ./cmd/api

import: check-env
	@set -a; . ./$(ENV_FILE); set +a; DATABASE_URL="$${ADMIN_DATABASE_URL:-$$DATABASE_URL}"; export DATABASE_URL; cd apps/backend && exec go run ./cmd/import -corpus ../../docs/repository-data

embed: check-env
	@set -a; . ./$(ENV_FILE); set +a; DATABASE_URL="$${ADMIN_DATABASE_URL:-$$DATABASE_URL}"; export DATABASE_URL; cd apps/backend && exec go run ./cmd/embed

backend-test: check-env
	@set -a; . ./$(ENV_FILE); set +a; cd apps/backend && go vet ./... && go test ./...

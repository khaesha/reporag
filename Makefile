SHELL := /bin/sh
ENV_FILE := .env

.PHONY: env check-env db-up db-down api import backend-test

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

api: check-env
	@set -a; . ./$(ENV_FILE); set +a; cd apps/backend && exec go run ./cmd/api

import: check-env
	@set -a; . ./$(ENV_FILE); set +a; cd apps/backend && exec go run ./cmd/import -corpus ../../docs/repository-data

backend-test: check-env
	@set -a; . ./$(ENV_FILE); set +a; cd apps/backend && go vet ./... && go test ./...

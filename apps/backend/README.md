# SearchLens backend

## Requirements

- Go 1.27+
- Docker with Compose

With GVM:

```sh
gvm install go1.27.1 -B
gvm use go1.27.1
```

Set `POSTGRES_DB`, `POSTGRES_USER`, and `POSTGRES_PASSWORD` in your shell, then start PostgreSQL from the repository root:

```sh
docker compose up -d db
```

The initial migration runs when Compose creates a fresh database volume. For an existing database, apply it explicitly:

```sh
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/001_init.sql
```

Set the API connection string without committing it:

```sh
export DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT:-5432}/${POSTGRES_DB}?sslmode=disable"
go run ./cmd/api
```

Optional configuration:

| Variable | Default |
| --- | --- |
| `PORT` | `8080` |
| `FRONTEND_ORIGIN` | `http://localhost:3000` |
| `REQUEST_TIMEOUT` | `10s` |

Probe the process and database:

```sh
curl -i http://localhost:8080/healthz
curl -i http://localhost:8080/readyz
```

Run checks from this directory:

```sh
gofmt -w .
go vet ./...
go test ./...
```

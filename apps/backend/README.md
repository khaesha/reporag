# SearchLens backend

## Requirements

- Go 1.27+
- Docker with Compose

With GVM:

```sh
gvm install go1.27.1 -B
gvm use go1.27.1
```

Create the local server environment from the repository root:

```sh
make env
```

Edit the generated `.env` and set `POSTGRES_PASSWORD`. Keep values compatible with shell syntax and URL-encode reserved characters used in `DATABASE_URL`. The file is ignored by Git and created with mode `0600`.

Start PostgreSQL:

```sh
make db-up
```

The initial migration runs when Compose creates a fresh database volume.

Start the API:

```sh
make api
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

Import the corpus explicitly after the migration is applied:

```sh
make import
```

The command prints `inserted`, `updated`, `rejected`, and `total` counts. It updates existing records by URI, so repeated imports do not create duplicates. A malformed record rolls back its file and exits with its filename and zero-based array position.

Uncomment `TEST_DATABASE_URL` in `.env` to include the importer integration test against a disposable database. Then run:

```sh
make backend-test
```

Go commands use the active GVM version. Formatting still runs from this directory:

```sh
gofmt -w .
```

Stop local services from the repository root with `make db-down`. Frontend configuration will use `apps/frontend/.env.local`; database credentials never belong there.

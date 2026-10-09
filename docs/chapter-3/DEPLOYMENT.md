# Chapter 3 Local Deployment and Recovery

Status: target runbook; commands added during Chapter 3 must be marked verified only after successful execution.

## 1. Release claim

Chapter 3 proves a local release on one host. PostgreSQL remains loopback-bound, the Go API and Next.js frontend run locally, and external model calls use the existing server-side OpenRouter credential.

This runbook does not prove public or hosted production. Hosted deployment remains [To Be Discussed](../TECH_DEBT.md).

## 2. Prerequisites

- Docker with Compose.
- Go 1.27.1 through the repository's documented GVM workflow.
- Node.js 20.9 or newer and npm.
- PostgreSQL client tools compatible with the local server for dump and restore, or equivalent tools inside the database container.
- An ignored root `.env` created from `.env.example`.
- `OPENROUTER_API_KEY` for embedding, semantic search, and synthesis checks.

Never commit `.env`, dumps, passwords, provider keys, or full generated review content containing sensitive input.

## 3. Database connection

Chapter 3 keeps the existing environment contract:

- `DATABASE_URL`: local connection used by migration, import, embedding, and API commands.
- `TEST_DATABASE_URL`: optional disposable test connection for database integration tests.
- `POSTGRES_DB`, `POSTGRES_USER`, and `POSTGRES_PASSWORD`: Docker database initialization.

Keep this single local connection contract for Chapter 3.

## 4. Existing bootstrap path

Create local environment configuration, start PostgreSQL, apply migrations, import, embed, and print the database report:

```sh
rtk make env
rtk make db-up
rtk make migrate
rtk make import
rtk make embed
rtk make migrate-report
```

Run migration, import, and embedding a second time. The second import and embedding runs must report no changes. Record output in Chapter 3 evidence without secrets or full queries.

Do not destroy an existing volume to simulate a clean release. Use the verified Phase 3 command below; it creates two uniquely named disposable databases on the loopback PostgreSQL server and never targets the active database.

## 5. Start application

Start the API with the local `DATABASE_URL`:

```sh
rtk make api
```

Start the frontend from `apps/frontend`:

```sh
rtk npm ci
rtk npm run dev
```

Open `http://localhost:3000`. PostgreSQL must remain bound to `127.0.0.1`; do not expose port 5432 publicly.

## 6. Local scope guard

Keep PostgreSQL bound to `127.0.0.1`. Run readiness and representative API requests through the existing local connection. Public exposure, TLS, proxy configuration, and hosted access controls are outside Chapter 3.

## 7. Backup and restore

Create a custom-format dump through the existing local connection and save it outside tracked repository files. The verified command builds a clean disposable database, imports and embeds twice, dumps it, restores into a second disposable database, and writes safe evidence under `docs/chapter-3`:

```sh
rtk make phase3-verify BACKUP_DIR=/absolute/path/outside/repository
```

`BACKUP_DIR` must already exist and be absolute. Retain the emitted custom-format dump as the last verified local recovery artifact. The command rejects non-loopback database URLs, uses generated `searchlens_phase3_*` names only, stops its temporary API, and drops only those generated databases. Never use `--clean` against the active release database.

After restore, verify:

- PostgreSQL and pgvector versions;
- schema and migration availability;
- 2,190 documents and 2,190 embeddings for the unchanged Chapter 2 corpus;
- API readiness;
- one lexical, semantic, hybrid, related, trend, and synthesis request.

The verified Phase 3 result is recorded in [`database-operations.json`](database-operations.json). If the command fails, it still attempts to drop only its generated disposable databases; the dump remains in `BACKUP_DIR` if it was already created.

## 8. Rollback and cleanup

- Application rollback: stop API and frontend, switch to the previously accepted code revision, and start them against the last verified database.
- Database rollback: restore the last verified dump into a new disposable database, verify it, then change the server-side `DATABASE_URL`.
- Failed release environment: remove only the explicitly named disposable database or Compose project created for that run.
- Never delete the default or active volume as part of an automated check.

Frontend configuration does not change during database rollback.

## 9. Release evidence

Record these under `docs/chapter-3`:

| Evidence | Required result |
| --- | --- |
| Environment report | PostgreSQL, pgvector, corpus digest, documents, embeddings, models |
| Exact-title verification | Every valid URI passed or has approved disposition |
| Retrieval evaluation | Chapter 2 thresholds preserved |
| Latency | Database and provider timing separated; local retrieval p95 below 500 ms |
| Synthesis review | At least 30 prompts; at least 90% claim support; valid citations |
| Usage and cost | Model, input/output usage, latency, estimated cost |
| Idempotency | Second migration/import/embed runs make no changes |
| Backup/restore | Dump checksum, restored counts, smoke checks |
| Application checks | Backend, frontend, scripts, API, browser |

## 10. Hosted deployment

**To Be Discussed.** Do not reuse this local runbook as evidence for a public deployment.

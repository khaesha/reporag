# Synthesis rate limit

Do not publicly enable `POST /api/v1/answer` until the deployment proxy enforces this limit before traffic reaches the Go API:

```nginx
limit_req_zone $binary_remote_addr zone=searchlens_answer:10m rate=5r/m;

location = /api/v1/answer {
    limit_req zone=searchlens_answer burst=2 nodelay;
    proxy_pass http://searchlens_api;
}
```

Use the deployment platform's equivalent when NGINX is not the proxy. Apply the limit to the verified client IP; do not trust an unvalidated forwarding header.

## Supabase PostgreSQL

Supabase hosts the existing PostgreSQL schema. It is not a frontend dependency: browser code receives neither a Supabase URL nor a database credential.

Use the ignored root `.env` file:

```sh
DATABASE_URL='postgres://...@pooler-host:5432/postgres?sslmode=require'
```

Copy endpoints from the Supabase dashboard. Prefer a direct connection with a CA certificate and `sslmode=verify-full`. The current IPv4-only proof uses dashboard-provided Supavisor session mode with `sslmode=require`: traffic is encrypted, but certificate verification is a recorded limitation. The current pgx path uses prepared statements, so use a direct connection or Supavisor **session** mode only—never transaction pooling.

Run migrations, import, and embeddings through the administrative direct connection:

```sh
make migrate
make import
make embed
make migrate-report
```

The API uses only its runtime `DATABASE_URL`. Its pool has five connections, which must remain below the selected Supabase project's connection limit.

Provision the runtime role through the administrative connection. Give it login, database connect, `USAGE` on `public` and `extensions`, and `SELECT` on `public.documents`; do not grant write, DDL, role-management, or superuser privileges. Verify failed `INSERT` and `CREATE TABLE` attempts with that role before release.

Use direct connections for backups and recovery. Save dumps outside the repository; restore them only into a disposable project for the smoke check:

```sh
pg_dump --format=custom --file=searchlens.dump "$DATABASE_URL"
pg_restore --clean --if-exists --no-owner --dbname "$RESTORE_ADMIN_DATABASE_URL" searchlens.dump
```

Run remote retrieval timing after starting the API with its runtime URL. The `Server-Timing` response header and benchmark output separate retrieval from query-embedding latency:

```sh
SEARCH_MODE=hybrid node scripts/benchmark-search.mjs http://localhost:8080
```

Rollback changes only the server-side runtime `DATABASE_URL`; frontend configuration remains unchanged.

### Phase 5 administrative evidence

On 2026-09-16, the `reporag` Supabase project was validated through its Session Pooler in `ap-northeast-1`: PostgreSQL 17.6, pgvector 0.8.2, 2,190 documents, and 2,190 current embeddings. Migrations, corpus import, and an idempotent embedding rerun completed through the administrative connection. Readiness; lexical, semantic, and hybrid retrieval; related records; trends; grounded synthesis; and exact-title lookup passed against the remote database.

Five post-warm-up hybrid requests measured retrieval p50/p95 of 390.18/411.38 ms and model p50/p95 of 798.31/894.51 ms. Session Pooler latency exceeds the 300 ms local retrieval benchmark threshold and remains recorded evidence, not a hidden pass.

Temporary security debt: local `.env` uses the administrative Session Pooler credential for the API because a custom-role pooler URL is unavailable. Replace `DATABASE_URL` with the `searchlens_runtime` Session Pooler credential before any public deployment, then prove that role cannot write corpus data or alter schema. Backup/restore validation remains pending PostgreSQL client tools and a disposable Supabase project. Both are release-gate work, not permission to deploy publicly.

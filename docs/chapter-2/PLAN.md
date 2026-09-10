# Chapter 2 Delivery Plan

Each phase is a separate delivery unit. Before starting one, update local `main`, create a new phase branch from `main`, and confirm the previous phase was accepted and merged. Complete its checks, commit, push, and open a pull request; never merge it automatically.

## Phase 0: Lock corpus, models, and evaluation

- [ ] Finish collecting and audit at least 2,000 valid unique records.
- [ ] Report files, records, unique URIs, years, degree programs, missing fields, invalid fields, duplicate URIs, and probable duplicate titles.
- [ ] Review deterministic normalization mappings for divisions, item types, and subjects; preserve original source values.
- [ ] Expand the judged retrieval set to at least 100 queries covering every included program and lexical, conceptual, bilingual, abbreviation, exact-title, and exact-author intents.
- [ ] Allow multiple relevant URIs and graded relevance judgments per query.
- [ ] Record Chapter 1 lexical `Recall@10`, `MRR@10`, `nDCG@10`, top-5 success, and p95 on the expanded corpus.
- [ ] Benchmark candidate embedding models on a representative Indonesian-English subset.
- [ ] Lock one embedding model, dimensions, one generation model, API provider, token limits, timeout, and expected cost.
- [ ] Document that generated output uses metadata and available abstracts only.

Checks:

```sh
node scripts/audit-corpus.mjs
node --test scripts/audit-corpus.test.mjs
```

Add the smallest evaluation command needed to reproduce the new baseline.

Exit: corpus facts are reproducible, every degree program has judged queries, lexical baseline is saved, and model choices are explicit. Do not write the vector migration before dimensions are locked.

## Phase 1: Add embedding lifecycle

- [ ] Add pgvector to local PostgreSQL and apply one additive migration.
- [ ] Add embedding, model, input-hash, and timestamp columns to `documents`.
- [ ] Implement canonical embedding input from existing record fields.
- [ ] Add one concrete model API client with response-dimension validation, timeouts, redacted errors, and bounded retry for transient failures.
- [ ] Add `cmd/embed` to process missing or stale rows in bounded batches.
- [ ] Skip unchanged rows when input hash and model identifier match.
- [ ] Preserve metadata and lexical search when any embedding fails.
- [ ] Print embedded, skipped, failed, and total counts.
- [ ] Add deterministic tests using a fake HTTP model server.

Checks from `apps/backend`:

```sh
gofmt -w .
go vet ./...
go test ./...
```

Operational check: import the full corpus, embed it twice, and confirm the second run performs no embedding requests for unchanged rows.

Exit: every eligible record has a current valid vector or an explicit reported failure; repeated runs are idempotent.

## Phase 2: Build and evaluate hybrid retrieval

- [ ] Extend `GET /api/v1/search` with `lexical`, `semantic`, and `hybrid` modes; default to hybrid.
- [ ] Reuse all Chapter 1 filters, validation, alternate sorts, pagination, and response fields.
- [ ] Retrieve fixed lexical and vector candidate sets and fuse them with reciprocal rank fusion.
- [ ] Preserve exact-title and exact-author ranking.
- [ ] Return lexical results when query embedding fails in hybrid mode; return a useful error in semantic mode.
- [ ] Keep records without vectors discoverable lexically.
- [ ] Add filtered semantic, fusion-ordering, stable-pagination, fallback, and regression tests.
- [ ] Run lexical, semantic, and hybrid evaluation on the same judged set.
- [ ] Measure exact vector-search p95; do not add HNSW unless it misses the latency gate.

Checks from `apps/backend`:

```sh
gofmt -w .
go vet ./...
go test ./...
```

Exit: hybrid improves `nDCG@10` by at least 10% relative on conceptual and bilingual queries, exact lookup does not regress, at least 85% of all queries have a relevant top-5 result, and local hybrid-search p95 is below 500 ms.

## Phase 3: Add related theses and corpus trends

- [ ] Add `GET /api/v1/related` using stored embeddings and optional division filtering.
- [ ] Exclude the source record and distinguish missing URI from unavailable embedding.
- [ ] Add `GET /api/v1/trends` using parameterized SQL aggregations.
- [ ] Return counts by year, division, item type, normalized subject, and missing abstract.
- [ ] Bound trend result sizes and report missing values without inference.
- [ ] Add frontend related-thesis actions and a corpus-trends view.
- [ ] Describe trends as indexed corpus coverage, not total institutional output.
- [ ] Preserve loading, empty, error, keyboard, focus, touch-target, and text-equivalent behavior.

Backend checks:

```sh
gofmt -w .
go vet ./...
go test ./...
```

Frontend checks from `apps/frontend`:

```sh
npm run lint
npx tsc --noEmit
npm exec -- next build --webpack
```

Exit: users can move from one record to related records and inspect reproducible corpus trends without generated labels.

## Phase 4: Add grounded synthesis

- [ ] Add `POST /api/v1/answer` with strict body and filter validation.
- [ ] Retrieve context server-side through hybrid search; reject client-provided evidence.
- [ ] Use at most eight records with abstracts under a fixed token budget.
- [ ] Treat metadata and abstracts as untrusted evidence and delimit them from model instructions.
- [ ] Request structured output with numbered citation identifiers.
- [ ] Verify every citation identifier against retrieved context before returning output.
- [ ] Return explicit insufficient-evidence state and abstract-only basis label.
- [ ] Bound model concurrency, request time, input size, and output size.
- [ ] Log latency, usage, model, and estimated cost without full queries, evidence, or answers.
- [ ] Add explicit frontend synthesis action, cited links, and separate loading, insufficient-evidence, failure, and success states.
- [ ] Add deterministic fake-model tests for supported, insufficient, malformed, invalid-citation, timeout, and provider-error responses.
- [ ] Configure deployment-level rate limits before enabling the endpoint publicly.

Backend checks:

```sh
gofmt -w .
go vet ./...
go test ./...
```

Frontend checks from `apps/frontend`:

```sh
npm run lint
npx tsc --noEmit
npm exec -- next build --webpack
```

Exit: generated text is optional, abstract-grounded, citation-validated, cost-bounded, and unable to break ordinary retrieval.

## Phase 5: Prove Supabase deployment

- [ ] Create a clean Supabase project in the intended production region and record its PostgreSQL and pgvector versions.
- [ ] Keep `apps/backend/migrations` as the single migration source and add the smallest command that applies every migration in order through a direct connection.
- [ ] Ensure the pgvector migration uses the `extensions` schema and works unchanged in local PostgreSQL and Supabase.
- [ ] Provision separate administrative and least-privilege runtime database credentials; never expose either to the frontend.
- [ ] Require remote TLS and document certificate verification or its recorded limitation.
- [ ] Use a direct runtime connection when supported; otherwise use Supavisor session mode for an IPv4-only backend.
- [ ] Do not use transaction pooling with current pgx prepared statements.
- [ ] Apply migrations, import the final corpus, and generate embeddings through the direct administrative connection.
- [ ] Configure the deployed Go API with only its runtime `DATABASE_URL` and existing server/model secrets.
- [ ] Set a bounded pgx pool below the selected Supabase project's connection limit.
- [ ] Run readiness, lexical, semantic, hybrid, related, trend, and synthesis smoke checks against Supabase.
- [ ] Verify exact-title searchability and record count match the clean local baseline.
- [ ] Verify the runtime role cannot alter schema or write corpus records.
- [ ] Run `pg_dump` and restore smoke checks through direct connections.
- [ ] Measure Supabase retrieval p50 and p95 separately from model latency.
- [ ] Document rollback by restoring the previous server-side `DATABASE_URL`; no frontend change should be required.

Checks:

```sh
make backend-test
```

Also run the documented remote migration, import, embedding, backup, restore, and API smoke commands. Never place Supabase credentials in committed files or command output.

Exit: unchanged Go application and migrations work against Supabase PostgreSQL, pgvector queries pass, privileges are limited, secrets stay server-side, and remote measurements are recorded.

## Phase 6: Chapter 2 release gate

- [ ] Import and audit the final corpus in a clean database.
- [ ] Confirm exact-title searchability for every valid unique URI.
- [ ] Complete embeddings with no unreported failures; confirm idempotent rerun.
- [ ] Re-run lexical, semantic, and hybrid evaluation and record all metrics.
- [ ] Manually judge at least 30 synthesis prompts and calculate claim support.
- [ ] Confirm at least 90% of factual generated claims are supported by cited retrieved abstracts.
- [ ] Confirm every returned citation maps to retrieved evidence and its repository URI.
- [ ] Measure lexical, semantic, hybrid, related, trend, and model latency separately.
- [ ] Record embedding and generation usage and cost for reproducible test workloads.
- [ ] Run backend, frontend, corpus audit, migration, backup, restore, and browser smoke checks.
- [ ] Confirm final production smoke checks pass against Supabase using the least-privilege runtime connection.
- [ ] Update root README with Chapter 2 setup, model configuration, embedding, evaluation, limitations, and rollback steps.
- [ ] Record release environment, corpus facts, model identifiers, retrieval metrics, local and Supabase latency, synthesis support rate, and cost below.

Exit: all Chapter 2 PRD success criteria pass and evidence is recorded. User reviews and merges the release pull request manually.

### Release evidence

Pending implementation and measurement.

## Deferred until evidence requires it

- HNSW vector index.
- Cross-encoder reranking.
- Document chunking.
- Background embedding workers.
- Separate vector database or model service.
- Multiple-provider abstraction.
- GraphRAG, fine-tuning, and autonomous agents.

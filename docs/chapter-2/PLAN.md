# Chapter 2 Delivery Plan

Each phase is a separate delivery unit. Before starting one, update local `main`, create a new phase branch from `main`, and confirm the previous phase was accepted and merged. Complete its checks, commit, push, and open a pull request; never merge it automatically.

## Phase 0: Lock corpus, models, and evaluation

- [x] Finish collecting and audit at least 2,000 valid unique records.
- [x] Report files, records, unique URIs, years, degree programs, missing fields, invalid fields, duplicate URIs, and probable duplicate titles.
- [x] Review deterministic normalization mappings for divisions, item types, and subjects; preserve original source values.
- [x] Expand the judged retrieval set to at least 100 queries covering every included program and lexical, conceptual, bilingual, abbreviation, exact-title, and exact-author intents.
- [x] Allow multiple relevant URIs and graded relevance judgments per query.
- [x] Record Chapter 1 lexical `Recall@10`, `MRR@10`, `nDCG@10`, top-5 success, and p95 on the expanded corpus.
- [x] Benchmark candidate embedding models on the full representative Indonesian-English corpus.
- [x] Lock one embedding model, dimensions, one generation model, API provider, token limits, timeout, and expected cost.
- [x] Document that generated output uses metadata and available abstracts only.

Checks:

```sh
node scripts/audit-corpus.mjs
node --test scripts/audit-corpus.test.mjs
```

Run `node scripts/evaluate-search.mjs` against the Chapter 1 API to reproduce the lexical baseline. Run `node scripts/benchmark-embeddings.mjs` with `OPENROUTER_API_KEY` after the judged set is approved.

Current status: Phase 0 accepted. Corpus, audit, normalization mappings, judged evaluation, lexical baseline, OpenRouter provider, embedding benchmark, and model locks are recorded.

Exit: corpus facts are reproducible, every degree program has judged queries, lexical baseline is saved, and model choices are explicit. Do not write the vector migration before dimensions are locked.

## Phase 1: Add embedding lifecycle

- [x] Add pgvector to local PostgreSQL and apply one additive migration.
- [x] Add embedding, model, input-hash, dimension, and timestamp columns to `documents`.
- [x] Implement canonical embedding input from existing record fields.
- [x] Add one concrete model API client with response-dimension validation, timeouts, redacted errors, and bounded retry for transient failures.
- [x] Add `cmd/embed` to process missing or stale rows in bounded batches.
- [x] Skip unchanged rows when input hash and model identifier match.
- [x] Preserve metadata and lexical search when any embedding fails.
- [x] Print embedded, skipped, failed, and total counts.
- [x] Add deterministic tests using a fake HTTP model server.

Checks from `apps/backend`:

```sh
gofmt -w .
go vet ./...
go test ./...
```

Operational check: import the full corpus, embed it twice, and confirm the second run performs no embedding requests for unchanged rows.

Current status: complete. Local pgvector migration applied; 2,190 imported unique-URI records embedded with `google/gemini-embedding-2` at 1,536 dimensions. The second run reported `embedded=0 skipped=2190 failed=0 total=2190`.

Exit: every eligible record has a current valid vector or an explicit reported failure; repeated runs are idempotent.

## Phase 2: Build and evaluate hybrid retrieval

- [x] Extend `GET /api/v1/search` with `lexical`, `semantic`, and `hybrid` modes; default to hybrid.
- [x] Reuse all Chapter 1 filters, validation, alternate sorts, pagination, and response fields.
- [x] Retrieve fixed lexical and vector candidate sets and fuse them with reciprocal rank fusion.
- [x] Preserve exact-title and exact-author ranking.
- [x] Return lexical results when query embedding fails in hybrid mode; return a useful error in semantic mode.
- [x] Keep records without vectors discoverable lexically.
- [x] Add filtered semantic, fusion-ordering, stable-pagination, fallback, and regression tests.
- [x] Run lexical, semantic, and hybrid evaluation on the same judged set.
- [x] Measure exact vector-search p95; do not add HNSW unless it misses the latency gate.

Checks from `apps/backend`:

```sh
gofmt -w .
go vet ./...
go test ./...
```

Exit: hybrid improves `nDCG@10` by at least 10% relative on conceptual and bilingual queries, exact lookup does not regress, at least 85% of all queries have a relevant top-5 result, and local hybrid-search p95 is below 500 ms.

Current status: complete. Hybrid nDCG@10 is 0.8559 for conceptual queries and 0.9347 for bilingual queries, versus lexical 0 and 0.2; exact title and author success is 1.0; overall top-5 success is 0.99. Exact local vector-search p95 is 18.1 ms, so no HNSW index is needed.

## Phase 3: Add related theses and corpus trends

- [x] Add `GET /api/v1/related` using stored embeddings and optional division filtering.
- [x] Exclude the source record and distinguish missing URI from unavailable embedding.
- [x] Add `GET /api/v1/trends` using parameterized SQL aggregations.
- [x] Return counts by year, division, item type, normalized subject, and missing abstract.
- [x] Bound trend result sizes and report missing values without inference.
- [x] Add frontend related-thesis actions and a corpus-trends view.
- [x] Describe trends as indexed corpus coverage, not total institutional output.
- [x] Preserve loading, empty, error, keyboard, focus, touch-target, and text-equivalent behavior.

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

Current status: complete. Related retrieval uses current stored embeddings only, excludes the source, and distinguishes missing source from unavailable embedding. Trends use parameterized metadata aggregations, audited primary subject classes, and bounded result sets.

## Phase 4: Add grounded synthesis

- [x] Add `POST /api/v1/answer` with strict body and filter validation.
- [x] Retrieve context server-side through hybrid search; reject client-provided evidence.
- [x] Use at most eight records with abstracts under a fixed token budget.
- [x] Treat metadata and abstracts as untrusted evidence and delimit them from model instructions.
- [x] Request structured output with numbered citation identifiers.
- [x] Verify every citation identifier against retrieved context before returning output.
- [x] Return explicit insufficient-evidence state and abstract-only basis label.
- [x] Bound model concurrency, request time, input size, and output size.
- [x] Log latency, usage, model, and estimated cost without full queries, evidence, or answers.
- [x] Add explicit frontend synthesis action, cited links, and separate loading, insufficient-evidence, failure, and success states.
- [x] Add deterministic fake-model tests for supported, insufficient, malformed, invalid-citation, timeout, and provider-error responses.
- [x] Configure deployment-level rate limits before enabling the endpoint publicly.

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

Current status: complete. Synthesis retrieves hybrid evidence server-side, sends only bounded available abstracts to the locked generation model, rejects malformed or unsupported citations, and remains disabled for public traffic until the documented proxy rate limit is installed.

## Phase 5: Prove Supabase deployment

- [x] Create a clean Supabase project in the intended production region and record its PostgreSQL and pgvector versions.
- [x] Keep `apps/backend/migrations` as the single migration source and add the smallest command that applies every migration in order through a direct connection.
- [x] Ensure the pgvector migration uses the `extensions` schema and works unchanged in local PostgreSQL and Supabase.
- [ ] Provision separate administrative and least-privilege runtime database credentials; never expose either to the frontend.
- [ ] Require remote TLS and document certificate verification or its recorded limitation.
- [ ] Use a direct runtime connection when supported; otherwise use Supavisor session mode for an IPv4-only backend.
- [ ] Do not use transaction pooling with current pgx prepared statements.
- [ ] Apply migrations, import the final corpus, and generate embeddings through the direct administrative connection.
- [ ] Configure the deployed Go API with only its runtime `DATABASE_URL` and existing server/model secrets.
- [x] Set a bounded pgx pool below the selected Supabase project's connection limit.
- [x] Run readiness, lexical, semantic, hybrid, related, trend, and synthesis smoke checks against Supabase.
- [x] Verify exact-title searchability and record count match the clean local baseline.
- [ ] Verify the runtime role cannot alter schema or write corpus records.
- [ ] Run `pg_dump` and restore smoke checks through direct connections.
- [x] Measure Supabase retrieval p50 and p95 separately from model latency.
- [x] Document rollback by restoring the previous server-side `DATABASE_URL`; no frontend change should be required.

Checks:

```sh
make backend-test
```

Also run the documented remote migration, import, embedding, backup, restore, and API smoke commands. Never place Supabase credentials in committed files or command output.

Exit: unchanged Go application and migrations work against Supabase PostgreSQL, pgvector queries pass, privileges are limited, secrets stay server-side, and remote measurements are recorded.

Current status: accepted with explicit tech debt. The Supabase Session Pooler project has PostgreSQL 17.6, pgvector 0.8.2, 2,190 documents, and 2,190 embeddings; migrations and remote API smoke checks pass. Deferred to the Chapter 2 release gate: repeat administrative operations through direct IPv6 or an IPv4 add-on; replace the temporary administrative API credential with `searchlens_runtime` and prove it cannot write or alter schema; run backup/restore through PostgreSQL client tools against a disposable project. Do not deploy publicly until all are resolved.

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

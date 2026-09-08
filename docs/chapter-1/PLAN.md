# Chapter 1 Delivery Plan

The plan is ordered so that every phase leaves a runnable, verifiable result.

## Phase 0: Lock the baseline

- [x] Add a corpus audit command or test that reports file count, record count, degree programs, missing fields, and duplicate URIs.
- [x] Save at least 30 representative search queries with manually expected records; expand the set when a degree program is added.
- [x] Agree on the Chapter 1 wording: `Search thesis metadata and available abstracts`, not full-document Q&A.

Run `node scripts/audit-corpus.mjs` from the repository root to reproduce the corpus baseline. Run `node --test scripts/audit-corpus.test.mjs` to check the audit and evaluation set.

Exit: corpus numbers are reproducible and the evaluation set exists.

## Phase 1: Bootstrap the backend

- [x] Initialize the Go module under `apps/backend`.
- [x] Add Gin and pgx only.
- [x] Add configuration for database URL, frontend origin, port, and request timeout.
- [x] Implement graceful startup and shutdown.
- [x] Add `/healthz` and `/readyz`.
- [x] Add Docker Compose PostgreSQL and the initial migration.

Check: the service starts, readiness fails without PostgreSQL, and succeeds with it.

## Phase 2: Import the corpus

- [x] Define the typed source record and database record.
- [x] Recursively discover yearly JSON files, derive `source_year` from filenames, and retain degree program in `divisions`.
- [x] Normalize fields and parse deposit dates.
- [x] Upsert records by URI in one transaction per file.
- [x] Print inserted, updated, rejected, and total counts.
- [x] Add one integration check proving a second import leaves the record count unchanged.

Check: every valid unique record in the configured corpus imports; malformed input identifies its exact location.

## Phase 2.5: Add the local environment workflow

- [x] Add an ignored root `.env` with a committed, secret-free template for PostgreSQL and backend settings.
- [x] Add Make targets to create the file once, validate required values, run PostgreSQL, start the API, import the corpus, and run backend tests.
- [x] Document the GVM and Make workflow without adding a dotenv dependency.
- [x] Reserve `apps/frontend/.env.local` for Phase 4 public frontend settings; never put database credentials there.

Check: one root `.env` drives Compose and Go commands, stays untracked, and a repeated setup does not overwrite local values.

## Phase 3: Build retrieval

- [x] Implement full-text search across weighted metadata.
- [x] Add year, division, item-type, and abstract-availability filters.
- [x] Add relevance, title, and date sorts.
- [x] Add pagination and filter-value endpoints.
- [x] Validate query length, page, limit, filters, and sort values.
- [x] Test exact title, author, no-result, filter, sorting, and pagination cases.

Check: at least 24 of the 30 evaluation queries return a relevant result in the top 5.

## Phase 4: Connect the frontend

- [ ] Replace the timeout and hard-coded result array with the search API.
- [ ] Render loading, results, empty, and error states.
- [ ] Add filters, sort controls, and working pagination.
- [ ] Link each result to its repository URI.
- [ ] Label missing abstracts and remove unsupported RAG/LLM claims.
- [ ] Preserve keyboard and screen-reader behavior.

Check: a user can search, filter, paginate, and open a source record without a console or accessibility error.

## Phase 5: Release gate

- [ ] Run Go tests and frontend lint/build.
- [ ] Import the full corpus into a clean database.
- [ ] Confirm all PRD acceptance criteria.
- [ ] Measure local search p95 against the 30-query evaluation set.
- [ ] Document setup, import, run, test, and backup commands.

Exit: Chapter 1 is deployable and its retrieval measurements are recorded.

## Recommended implementation order

Implement Phases 1 and 2 first. Do not begin frontend integration until the search response contract and importer are stable. Do not begin Chapter 2 until the evaluation set exposes where keyword retrieval actually fails.

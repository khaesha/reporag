# Chapter 2 Review and Chapter 3–4 Handoff

Status: merged functional release candidate; not approved for public production  
Reviewed: 2026-10-08  
Merged range: PRs #9–#15, through commit `5213e61`

## Executive summary

Chapter 2 delivered semantic and hybrid thesis discovery, related-thesis retrieval, corpus trends, grounded abstract synthesis, an idempotent embedding lifecycle, and Supabase compatibility. All planned feature phases are merged into `main`.

Retrieval goals passed. Hybrid search materially improves conceptual and bilingual discovery, preserves exact-title and exact-author evaluation cases, and places a relevant result in the top five for 99% of the 100-query judged set.

Chapter 2 is not production-complete against its original hosted release gate. Remaining quality and operational work moves into Chapter 3 with local PostgreSQL as the acceptance environment. Hosted deployment is To Be Discussed. Previous corpus-expansion recommendation moves to Chapter 4.

## Delivered

| Area | Result |
| --- | --- |
| Corpus | 42 files, 2,191 source rows, 2,190 unique URIs, 2,187 valid non-quarantined URIs across three degree programs |
| Evaluation | 100 graded queries covering exact-title, exact-author, lexical, conceptual, bilingual, and abbreviation intents |
| Embeddings | `google/gemini-embedding-2`, 1,536 dimensions; 2,190 stored vectors; unchanged rerun skipped all 2,190 records |
| Search | `lexical`, `semantic`, and default `hybrid` modes with existing filters, sorts, pagination, and lexical fallback |
| Related records | Stored-vector similarity, source exclusion, optional division filter, bounded result count |
| Trends | Bounded metadata counts by year, division, item type, normalized subject, and missing abstract |
| Synthesis | Explicit `POST /api/v1/answer`, server-selected evidence, abstract-only grounding, checked citation identifiers, bounded context and concurrency |
| Deployment | Same Go API and ordered migrations tested against Supabase PostgreSQL 17.6 with pgvector 0.8.2 through Session Pooler |
| Operations | Migration, import, embedding, evaluation, title-verification, synthesis-evaluation, and search-benchmark commands documented |

## Quality evidence

### Release retrieval metrics

| Mode | Recall@10 | MRR@10 | nDCG@10 | Top-5 success | End-to-end p50 / p95 |
| --- | ---: | ---: | ---: | ---: | ---: |
| Lexical | 0.6100 | 0.5950 | 0.5989 | 0.6100 | 195.36 / 217.67 ms |
| Semantic | 0.9767 | 0.9198 | 0.9274 | 0.9900 | 850.29 / 1,140.54 ms |
| Hybrid | 0.9767 | 0.9503 | 0.9503 | 0.9900 | 1,033.93 / 1,320.58 ms |

Semantic and hybrid timings include remote query-embedding time. Exact local vector-search p95 was 18.1 ms, so an approximate vector index is not justified.

Hybrid subset nDCG@10 reached 0.8559 for conceptual queries and 0.9347 for bilingual queries, compared with lexical scores of 0 and 0.2. Judged exact-title and exact-author success remained 1.0.

### Model and cost decisions

- Embedding: `google/gemini-embedding-2`; strongest benchmark quality at 0.9040 conceptual-plus-bilingual nDCG@10.
- Generation: `openai/gpt-5.6-luna`; 16,000-token input budget, 800-token output limit, 30-second timeout, maximum concurrency four.
- Recorded configured maximum generation cost: about $0.00416 per answer at benchmarked prices.
- `openai/text-embedding-3-small` remains a lower-cost fallback, not an active model.

## Planning decision

- Chapter 2 remains a merged functional release candidate. Historical evidence and unchecked gates stay unchanged in its plan.
- Chapter 3 inherits the unfinished Chapter 2 gates that can be proved locally.
- Chapter 3 uses Docker Compose PostgreSQL with pgvector as its release database.
- Chapter 3 claims local release readiness only.
- Hosted deployment is To Be Discussed and has no Chapter 3 requirements.
- Corpus expansion and incremental refresh, previously recommended for Chapter 3, move to Chapter 4.

## Chapter 3: Complete the local release

Goal: close retrieval, synthesis, database-operation, and end-to-end evidence gaps using the existing local stack. No new user-facing feature is required.

### Phase 0: Re-baseline the release target

- [ ] Record local PostgreSQL and pgvector versions, corpus digest, record count, embedding count, and active model identifiers.
- [ ] State that Chapter 3 release means a reproducible local release.
- [ ] Preserve Chapter 2 evidence without making it a Chapter 3 requirement.
- [ ] Record hosted deployment as `To Be Discussed`.

Exit: local acceptance environment and claims are unambiguous.

References: [Chapter 2 release gate](chapter-2/PLAN.md#phase-6-chapter-2-release-gate), [deployment evidence](chapter-2/DEPLOYMENT.md), [production tech debt](TECH_DEBT.md), and [root operating guide](../README.md#run-locally).

### Phase 1: Close retrieval correctness

- [ ] Classify all 93 exact-title misses: duplicate title, lexical ranking, malformed source value, verifier assumption, or product defect.
- [ ] Fix product defects at the shared retrieval layer; do not add per-record exceptions.
- [ ] Define expected behavior for distinct URIs sharing the same title.
- [ ] Rerun the full 2,187-record exact-title verifier and record final pass/failure counts.
- [ ] Rerun lexical, semantic, and hybrid evaluation against the unchanged 100-query set.
- [ ] Measure search, related, and trend latency locally as separate workloads.

Exit: every unresolved title miss has an evidence-backed disposition, retrieval quality does not regress, and local endpoint latency is recorded.

References: [exact-title results](chapter-2/release-exact-titles.json), [exact-title verifier](../scripts/verify-exact-titles.mjs), [judged query set](chapter-2/evaluation.json), [search evaluator](../scripts/evaluate-search.mjs), [search benchmark](../scripts/benchmark-search.mjs), and [release hybrid metrics](chapter-2/release-hybrid.json).

### Phase 2: Complete synthesis evidence

- [ ] Review every factual claim in the existing 30-prompt result set against its cited abstract.
- [ ] Fill each empty `claims` entry with the manual judgment evidence required by the evaluator format.
- [ ] Calculate claim-support rate and require at least 90% support.
- [ ] Confirm every returned citation maps to retrieved evidence and its authoritative repository URI.
- [ ] Record input/output token usage, model identifier, latency, and estimated cost for the reproducible workload.
- [ ] Rerun malformed-output, invalid-citation, insufficient-evidence, timeout, and provider-error tests.

Exit: human-reviewed synthesis evidence meets the Chapter 2 quality threshold and cost is reproducible.

References: [synthesis results](chapter-2/synthesis-results.json), [synthesis evaluator](../scripts/evaluate-synthesis.mjs), [judged query set](chapter-2/evaluation.json), [grounded-synthesis requirements](chapter-2/PRD.md#grounded-synthesis), and [PRD success criteria](chapter-2/PRD.md#10-success-criteria).

### Phase 3: Prove local database operations

- [ ] Recreate the local database from empty state using ordered migrations.
- [ ] Import the final corpus and generate embeddings; prove both reruns are idempotent.
- [ ] Create a local `pg_dump`, restore it into a disposable local database, and rerun readiness plus smoke checks.
- [ ] Document local rollback and recovery commands.

Exit: a clean local database can be built, backed up, restored, and verified through the existing local connection.

References: [initial migration](../apps/backend/migrations/001_init.sql), [pgvector migration](../apps/backend/migrations/002_semantic_search.sql), [migration command](../apps/backend/cmd/migrate/main.go), [import command](../apps/backend/cmd/import/main.go), [embedding command](../apps/backend/cmd/embed/main.go), [backend guide](../apps/backend/README.md), and [root backup guide](../README.md#database-backup).

### Phase 4: Local release gate

- [ ] Run backend formatting, vet, unit tests, and database integration tests.
- [ ] Run frontend tests, lint, TypeScript check, and production build.
- [ ] Run corpus audit and all script tests.
- [ ] Run readiness, lexical, semantic, hybrid, related, trend, and synthesis API smoke checks.
- [ ] Run browser smoke checks for loading, empty, error, keyboard, focus, related, trends, and synthesis states.
- [ ] Record final local environment, corpus facts, retrieval metrics, synthesis support, latency, usage, and cost.
- [ ] Update README and Chapter 3 evidence with exact reproduction commands and remaining limitations.

Exit: all locally applicable Chapter 2 release criteria pass from a clean environment. Release is labeled local-only.

References: [root checks](../README.md#checks), [Chapter 2 release checklist](chapter-2/PLAN.md#phase-6-chapter-2-release-gate), [backend instructions](../apps/backend/README.md), and [corpus audit](chapter-2/corpus-audit.json).

## Hosted deployment

**To Be Discussed.** No hosted provider, connection model, security contract, or production gate is selected for Chapter 3. Existing Chapter 2 [deployment evidence](chapter-2/DEPLOYMENT.md) remains historical only.

## Known data limitations

- Three anomalous URIs remain quarantined without changing raw source records.
- One duplicate URI and 34 probable duplicate-title groups require human review; no automatic merge or deletion is allowed.
- Three abstracts and five subject values are missing.
- Nine subject strings remain unmapped because no deterministic mapping was approved.
- Search and generation cover repository metadata and available abstracts only. Restricted PDFs and chapter-level content remain out of scope.

## Intentional deferrals

Do not carry these into Chapter 3 or Chapter 4 without evidence:

- HNSW or another approximate vector index: add only when exact search misses its latency gate at a larger measured corpus.
- Cross-encoder reranking: add only when judged retrieval quality falls below target and error analysis shows reranking would help.
- Document chunking: add only when legally available content becomes materially longer than abstracts.
- Background embedding workers: add only when explicit import-time embedding cannot meet freshness needs.
- Separate vector database, model service, provider abstraction, GraphRAG, fine-tuning, or autonomous agents: no current requirement.

## Chapter 4: Corpus expansion and incremental refresh

Start only after Chapter 3 local release gate passes.

Why this comes next:

- Retrieval quality already exceeds Chapter 2 targets; more ranking infrastructure has no measured need.
- Current coverage is limited to three degree programs, while the data model and UI already support division-generic search.
- Expanding useful, audited coverage increases product value without changing the proven architecture.

Proposed Chapter 4 scope:

1. Add more degree programs using the existing record contract and authoritative repository URIs.
2. Define a repeatable incremental collection/import process with provenance and duplicate review.
3. Re-run corpus audit, normalization review, embeddings, and exact-title verification after each addition.
4. Expand judged queries for every new program and compare lexical, semantic, and hybrid quality against the Chapter 2 baseline.
5. Add freshness and coverage reporting for maintainers; keep user-facing trend language limited to indexed corpus coverage.

Possible later product feature: a researcher workspace for selecting records, comparing available abstracts, and exporting citations. Define it only after Chapter 4 scope is approved; do not introduce accounts or synchronization unless persistence across devices is required.

## Chapter 4 entry checklist

- [ ] Complete Chapter 3 local release gate.
- [ ] Keep hosted deployment marked `To Be Discussed`.
- [ ] Approve Chapter 4 corpus sources and degree-program order.
- [ ] Write Chapter 4 PRD, architecture, and phased plan before implementation.
- [ ] Capture Chapter 2 metrics above as the immutable comparison baseline.

## Source documents

- [Chapter 3 PRD](chapter-3/PRD.md)
- [Chapter 3 architecture](chapter-3/ARCHITECTURE.md)
- [Chapter 3 delivery plan](chapter-3/PLAN.md)
- [Chapter 3 local deployment](chapter-3/DEPLOYMENT.md)
- [Chapter 2 PRD](chapter-2/PRD.md)
- [Chapter 2 architecture](chapter-2/ARCHITECTURE.md)
- [Chapter 2 delivery plan](chapter-2/PLAN.md)
- [Phase 0 evidence](chapter-2/PHASE-0.md)
- [Deployment evidence and runbook](chapter-2/DEPLOYMENT.md)
- [Production tech debt](TECH_DEBT.md)

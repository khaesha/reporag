# Chapter 3 Delivery Plan

Each phase is a separate delivery unit. Before starting one, update local `main`, create a new phase branch from `main`, and confirm the previous phase was accepted and merged. Complete relevant checks, commit, push, and open a pull request targeting `main`; never merge it automatically.

Chapter 3 targets a reproducible local release. Hosted deployment is [To Be Discussed](../TECH_DEBT.md).

## Phase 0: Lock the local release baseline

- [ ] Record local PostgreSQL and pgvector versions.
- [ ] Record corpus digest, source rows, unique URIs, valid URIs, database documents, and current embeddings.
- [ ] Record locked embedding and generation model identifiers and limits.
- [ ] Confirm Chapter 2 evidence is immutable and Chapter 3 evidence uses new files.
- [ ] Update root documentation to distinguish local release from any future hosted deployment.
- [ ] Record hosted deployment as `To Be Discussed`.

Checks:

```sh
rtk make db-up
rtk make migrate
rtk make migrate-report
rtk node scripts/audit-corpus.mjs
rtk node --test scripts/*.test.mjs
```

Exit: environment, corpus, models, evidence paths, and local-only claim are explicit and reproducible.

References: [PRD baseline](PRD.md#3-baseline), [architecture scope](ARCHITECTURE.md#2-scope-boundary), [cross-chapter review](../REVIEW.md), and [Chapter 2 evidence](../chapter-2/PLAN.md#release-evidence).

## Phase 1: Resolve exact-title verification and latency gaps

- [ ] Add the smallest diagnostic output needed to classify the 93 recorded misses.
- [ ] Identify duplicate-title groups, verifier defects, source anomalies, and retrieval defects.
- [ ] Define and test deterministic verification for distinct URIs sharing one title.
- [ ] Fix only confirmed shared retrieval or verifier defects.
- [ ] Rerun exact-title verification for all 2,187 valid URIs.
- [ ] Rerun lexical, semantic, and hybrid evaluation on the unchanged 100-query set.
- [ ] Measure lexical, semantic, hybrid, related, trend, vector, and provider latency separately.
- [ ] Save classification, verifier, evaluation, and latency evidence under `docs/chapter-3`.

Relevant checks:

```sh
rtk make backend-test
rtk node --test scripts/*.test.mjs
rtk node scripts/verify-exact-titles.mjs http://localhost:8080 docs/chapter-3/exact-titles.json
rtk node scripts/evaluate-search.mjs http://localhost:8080 docs/chapter-2/evaluation.json docs/chapter-3/hybrid-evaluation.json hybrid
```

Add or update commands only when required to produce related and trend latency evidence. Do not add a benchmark framework.

Exit: every valid URI has a passing or approved exact-title disposition, Chapter 2 retrieval thresholds do not regress, and local retrieval p95 remains below 500 ms.

References: [PRD exact-title requirements](PRD.md#82-exact-title-correctness), [architecture diagnosis](ARCHITECTURE.md#7-exact-title-diagnosis), [previous verifier output](../chapter-2/release-exact-titles.json), and [judged queries](../chapter-2/evaluation.json).

## Phase 2: Complete synthesis evaluation

- [ ] Define the minimal claim-review record shape.
- [ ] Review all factual claims across the existing 30-prompt workload against cited abstracts.
- [ ] Record support judgment and concise reviewer evidence for each claim.
- [ ] Calculate claim-support rate and require at least 90%.
- [ ] Confirm every citation maps to retrieved evidence and its authoritative URI.
- [ ] Capture generation model, input/output usage, latency, and estimated cost.
- [ ] Preserve supported, insufficient, malformed, invalid-citation, timeout, and provider-error tests.
- [ ] Save the completed review under `docs/chapter-3` without overwriting Chapter 2 output.

Checks:

```sh
rtk make backend-test
rtk node --test scripts/*.test.mjs
rtk node scripts/evaluate-synthesis.mjs http://localhost:8080 docs/chapter-3/synthesis-results.json
```

Exit: at least 30 prompts are manually judged, claim support is at least 90%, citations are valid, and usage and cost are reproducible.

References: [PRD synthesis requirements](PRD.md#84-synthesis-evidence), [architecture review data](ARCHITECTURE.md#8-synthesis-review-data), [Chapter 2 synthesis output](../chapter-2/synthesis-results.json), and [synthesis evaluator](../../scripts/evaluate-synthesis.mjs).

## Phase 3: Prove local database operations

- [ ] Build a clean isolated local database from ordered migrations.
- [ ] Import and embed twice; record idempotent second runs.
- [ ] Create a custom-format dump through the existing local connection.
- [ ] Restore into a separate disposable local database.
- [ ] Run readiness, representative API, count, and embedding checks after restore.
- [ ] Document safe cleanup and rollback without targeting the active database.

Checks:

```sh
rtk make backend-test
rtk make migrate-report
```

Phase implementation must add and document the smallest commands needed for backup and restore. Do not publish commands before they exist and pass.

Exit: clean build, idempotency, backup, restore, and restored-database smoke checks pass locally.

References: [PRD database requirements](PRD.md#85-local-database-operations), [database connection](ARCHITECTURE.md#4-database-connection), [backup architecture](ARCHITECTURE.md#9-backup-and-restore), and [deployment runbook](DEPLOYMENT.md).

## Phase 4: Chapter 3 local release gate

- [ ] Run all backend checks with database integration enabled.
- [ ] Run all frontend unit, lint, TypeScript, and production-build checks.
- [ ] Run corpus audit and all script tests.
- [ ] Run readiness, lexical, semantic, hybrid, related, trend, and synthesis smoke checks.
- [ ] Run browser checks for loading, empty, error, keyboard, focus, touch, related, trends, and synthesis states.
- [ ] Verify restored-database operation through the existing local connection.
- [ ] Record final environment, corpus, retrieval, latency, synthesis, usage, cost, backup, and restore evidence.
- [ ] Update README, this plan, and deployment documentation with exact verified commands.
- [ ] Confirm hosted deployment remains marked `To Be Discussed`.

Backend checks:

```sh
rtk make backend-test
```

Frontend checks from `apps/frontend`:

```sh
rtk npm test
rtk npm run lint
rtk npx tsc --noEmit
rtk npm exec -- next build --webpack
```

Script checks:

```sh
rtk node scripts/audit-corpus.mjs
rtk node --test scripts/*.test.mjs
```

Exit: every Chapter 3 success criterion passes from a clean local environment, evidence is committed, and documentation labels the release local-only. User reviews and merges the release pull request manually.

References: [PRD success criteria](PRD.md#9-success-criteria), [deployment evidence checklist](DEPLOYMENT.md#9-release-evidence), and [project tech debt](../TECH_DEBT.md).

## Deferred to Chapter 4 or later

- Corpus expansion and incremental refresh move to Chapter 4.
- Hosted deployment is To Be Discussed.
- HNSW, reranking, chunking, background workers, separate vector infrastructure, and provider abstraction still require measured need.

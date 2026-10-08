# Chapter 3 PRD: Reproducible Local Release

Status: Draft 1  
Owner: SearchLens team  
Scope: Chapter 3 only  
Decision date: 2026-10-02
Revised: 2026-10-08

## 1. Product summary

Chapter 3 completes the unfinished Chapter 2 release evidence against a local PostgreSQL and pgvector environment. It fixes or explains exact-title misses, completes human evaluation of grounded synthesis, proves backup recovery, and records one reproducible end-to-end local release.

Chapter 3 adds no user-facing feature. Existing Chapter 2 search, related-record, trend, and synthesis behavior remains the product under test.

## 2. Decision and source of truth

The [cross-chapter review](../REVIEW.md) moves locally provable Chapter 2 gates into Chapter 3. Hosted deployment is outside Chapter 3 and remains [To Be Discussed](../TECH_DEBT.md).

When documents disagree, use this order:

1. This PRD for Chapter 3 outcomes and acceptance.
2. [Chapter 3 architecture](ARCHITECTURE.md) for technical boundaries.
3. [Chapter 3 plan](PLAN.md) for implementation order.
4. [Chapter 3 deployment](DEPLOYMENT.md) for the local runbook.

Chapter 2 documents remain historical evidence and API contracts; do not rewrite their measured results.

## 3. Baseline

- Corpus: 42 files, 2,191 source rows, 2,190 unique URIs, and 2,187 valid non-quarantined URIs across three degree programs.
- Database: 2,190 imported documents and 2,190 current embeddings.
- Evaluation: 100 graded queries across exact-title, exact-author, lexical, conceptual, bilingual, and abbreviation intents.
- Retrieval: hybrid Recall@10 0.9767, MRR@10 0.9503, nDCG@10 0.9503, and top-5 success 0.9900.
- Title verification: 2,094 of 2,187 valid URIs passed; 93 require classification or correction.
- Synthesis: 30 prompts completed with structurally valid corpus citations; manual claim judgments and release-workload usage remain incomplete.

Baseline evidence lives in [Chapter 2](../chapter-2/PLAN.md#release-evidence).

## 4. Problem

Chapter 2 proved the feature set but did not close every release gate. Exact-title verification has unexplained failures, synthesis quality lacks human claim-level scoring, related and trend latency lack a local release measurement, and database recovery has not been proved from a clean environment.

Chapter 3 narrows the release claim to the existing local environment. Hosted deployment technology, security, and operations will be decided separately.

## 5. Users and primary jobs

### Maintainer

- Build the application from an empty local database using committed migrations and corpus files.
- Import and embed idempotently, measure retrieval, back up the database, and restore it safely.
- Diagnose every exact-title miss without record-specific ranking hacks.

### Reviewer

- Reproduce release metrics from committed scripts and evidence.
- Judge generated claims against cited available abstracts.
- Verify accessibility and failure states in a browser.
- Distinguish local release readiness from any future hosted-deployment claim.

## 6. Goals

- Account for every exact-title verification miss and fix shared product defects.
- Preserve Chapter 2 retrieval quality and API compatibility.
- Complete claim-level human review for the 30-prompt synthesis set.
- Record synthesis usage, cost, latency, and citation validity.
- Prove a clean local PostgreSQL and pgvector lifecycle: migrate, import, embed, operate, back up, restore, and verify.
- Record final backend, frontend, script, API, and browser checks.
- Produce evidence sufficient to call the project a local release.

## 7. Non-goals

- Public deployment or a production-readiness claim.
- Choosing or proving a hosted deployment target.
- New search modes, UI features, degree programs, or corpus collection.
- HNSW, reranking, chunking, background workers, a second database, or provider abstraction.
- Accessing restricted PDFs or generating claims beyond metadata and available abstracts.
- Changing model identifiers unless existing locked models become unavailable.

## 8. Requirements

### 8.1 Release baseline

1. Record PostgreSQL version, pgvector version, corpus digest, document count, embedding count, and active model identifiers from the local release environment.
2. The database record and embedding counts must match the unchanged Chapter 2 corpus evidence.
3. Commands and the API use the existing local `DATABASE_URL` contract.
4. Local database traffic must remain bound to loopback.

### 8.2 Exact-title correctness

1. Classify all 93 recorded failures as duplicate-title behavior, verifier defect, source anomaly, retrieval defect, or an explicitly approved limitation.
2. Fix retrieval defects in shared search or verifier logic; do not add URI-specific exceptions.
3. A full-title query for a unique valid title must return its URI within the first ten lexical results.
4. Distinct URIs with the same title must remain discoverable through deterministic pagination. The verifier must evaluate the title group rather than assume one URI owns the title.
5. Every one of the 2,187 valid non-quarantined URIs must receive a recorded pass or approved disposition.
6. Lexical, semantic, and hybrid evaluation must be rerun on the unchanged 100-query set after any search change.

### 8.3 Retrieval performance

1. Measure lexical, semantic, hybrid, related, trend, and query-embedding latency separately where applicable.
2. Local database retrieval p95 must remain below 500 ms for the fixed Chapter 2 corpus.
3. Provider time must not be reported as database retrieval time.
4. Hybrid top-5 success must remain at least 0.85, and exact-title and exact-author judged queries must not regress.

### 8.4 Synthesis evidence

1. Manually review at least 30 prompts covering supported and insufficient-evidence cases.
2. Break each supported answer into factual claims and judge every claim against its cited retrieved abstracts.
3. At least 90% of factual claims must be supported by cited evidence.
4. Every returned citation must match a retrieved corpus URI and title; invalid identifiers must remain rejected.
5. Record model identifier, input and output usage, latency, and estimated cost for the reproducible workload.
6. Preserve deterministic tests for supported, insufficient, malformed, invalid-citation, timeout, and provider-error responses.

### 8.5 Local database operations

1. Apply committed migrations in order to an empty disposable local database.
2. Import the final corpus and embed all eligible rows; repeated runs must make no data or embedding changes.
3. Create a custom-format backup and restore it into a separate disposable local database.
4. Run readiness and representative API smoke checks against the restored database.
5. Never restore over the active release database during verification.

### 8.6 End-to-end quality

1. Backend formatting, vet, unit, and database integration checks must pass.
2. Frontend unit, lint, TypeScript, and production-build checks must pass.
3. Corpus audit and script tests must pass.
4. Browser smoke checks must cover search, filters, related records, trends, synthesis, loading, empty, failure, keyboard, and visible-focus behavior.
5. Final evidence must state all remaining limitations and label the release local-only.

## 9. Success criteria

Chapter 3 passes when:

- all 2,187 valid URIs have a passing or approved exact-title disposition;
- Chapter 2 retrieval thresholds remain satisfied;
- local database retrieval p95 remains below 500 ms;
- at least 90% of reviewed synthesis claims are supported by cited abstracts;
- all generated citations in the review set map to retrieved authoritative URIs;
- synthesis usage and cost are recorded;
- migration, import, and embedding work from an empty database and are idempotent;
- backup and restore succeed into a disposable local database;
- backend, frontend, script, API, and browser checks pass; and
- documentation makes no public hosted-production claim.

## 10. Exit gate

Chapter 3 is complete when a reviewer can reproduce the full local release from committed code and data, verify its quality evidence, and restore its database. Hosted deployment remains [To Be Discussed](../TECH_DEBT.md) and does not block this local-only exit.

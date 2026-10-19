# SearchLens Tech Debt

Status: active cross-chapter source of truth  
Reviewed: 2026-10-19

## Chapter 3 closure debt

These accepted follow-ups do not invalidate the measured Chapter 3 retrieval, synthesis, backup, or restore results.

### Repeat import changes unchanged rows

Status: **Deferred to Chapter 4 Phase 0**

The Phase 3 second import reports 2,191 updates. `ImportFile` executes an update for every existing URI and refreshes `imported_at`, even when source values are identical. Counts and embeddings remain stable, but the rerun is not a database no-op.

Chapter 3 accepts the existing 2,190-document and 2,190-embedding database evidence without another import. Before Chapter 4 expands the corpus, update only rows whose imported fields differ, report zero updates for an unchanged corpus, and add one integration assertion for stable `imported_at`.

### Browser release-state evidence

Status: **Open manual follow-up; does not block Chapter 4**

The recorded browser pass exercised keyboard search, related theses, synthesis loading/success, and trends. Empty search was exercised only at the API, while empty, error/retry, visible-focus, and touch-target behavior was checked by source inspection.

Closure: manually run and record one browser pass that visibly exercises search and filter loading, empty, failure/retry, keyboard, focus, touch, related, trends, synthesis, and insufficient-evidence states. No new browser framework is required.

## Hosted deployment

Status: **To Be Discussed**

No provider, account tier, connection model, credential model, TLS policy, backup target, proxy, or production gate is selected. Define them in a future chapter before any public deployment claim.

This does not invalidate the Chapter 3 local release evidence. Chapter 2 [hosted deployment evidence](chapter-2/DEPLOYMENT.md) remains historical only. See the [cross-chapter review](REVIEW.md) and [Chapter 3 local deployment](chapter-3/DEPLOYMENT.md).

## Evidence-triggered deferrals

Do not schedule these without measured need:

- HNSW or another approximate vector index: current exact local retrieval p95 is below 500 ms.
- Cross-encoder reranking: hybrid quality remains above target.
- Document chunking: indexed content is metadata and available abstracts, not long full text.
- Background embedding workers: current explicit import/embed lifecycle meets the local release scope.
- Separate vector database, model service, provider abstraction, GraphRAG, fine-tuning, or autonomous agents: no current requirement.

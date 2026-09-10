# Chapter 2 PRD: Semantic Thesis Discovery and Grounded Synthesis

Status: Draft 1  
Owner: SearchLens team  
Scope: Chapter 2 only

## 1. Product summary

Chapter 2 helps users discover theses when their wording differs from repository metadata. It combines Chapter 1 full-text search with semantic retrieval, adds related-thesis and research-trend exploration, and can synthesize retrieved abstracts into a cited answer.

The product still searches only repository metadata and legally available abstracts. Generated text must identify its evidence and must never imply access to a restricted PDF or unsupported chapter-level content.

## 2. Corpus baseline

Chapter 1 shipped with 820 unique Computer Science records and a 30-query retrieval evaluation. Chapter 2 targets at least 2,000 valid unique records across available degree programs using the same source shape:

```text
title, abstract, authors, item_type, subjects, divisions,
depositing_user, date_deposited, uri
```

The actual Chapter 2 baseline must be recorded after collection finishes. Record count alone is not an exit gate: degree-program coverage, field completeness, duplicates, and evaluation coverage must also be reported.

## 3. Problem

Keyword search works well when a query repeats title, author, or abstract terms. It is weaker when users describe a concept with synonyms, mix Indonesian and English terminology, use abbreviations, or do not know the repository's wording.

Users must also open and compare many result cards themselves. A generated overview could reduce this work, but only if it is grounded in retrieved abstracts, cites authoritative repository records, and states when evidence is insufficient.

## 4. Users and primary jobs

### Student or researcher

- Find relevant theses from conceptual, bilingual, abbreviated, title, or author queries.
- Discover theses related to a promising record.
- Compare retrieved work by topic, approach, result, year, or degree program when abstracts provide that evidence.
- See corpus trends without treating them as claims about full thesis content.
- Open every cited repository record and inspect its available metadata.

### Maintainer

- Import new same-schema corpus files and embed new or changed records safely.
- Re-run embedding without recomputing unchanged records.
- Compare lexical, semantic, and hybrid retrieval on a fixed judged set.
- Track model identity, failures, latency, and generation cost.
- Replace a model through an explicit re-embedding migration, not silently.

## 5. Goals

- Improve conceptual and multilingual retrieval over the Chapter 1 lexical baseline.
- Preserve exact-title and exact-author behavior.
- Keep lexical, vector, and generated outputs traceable and independently testable.
- Add related-thesis discovery using the same document embeddings.
- Add factual corpus trends from stored metadata.
- Generate concise answers or comparisons from retrieved abstracts with verifiable citations.
- Establish repeatable quality, latency, and cost measurements for retrieval and generation.
- Run the same Go API and SQL schema against local PostgreSQL and a Supabase-hosted PostgreSQL production target.

## 6. Non-goals

- Downloading, parsing, indexing, or summarizing restricted PDFs.
- Claiming knowledge beyond supplied metadata and abstracts.
- Replacing repository URIs with generated or secondary links.
- A separate vector database, search cluster, model service, or microservice.
- Document chunking; each available abstract remains one retrieval unit.
- Fine-tuning, training a foundation model, GraphRAG, agent workflows, or autonomous research.
- Cross-encoder reranking unless hybrid retrieval fails the agreed evaluation gate.
- User accounts, saved searches, recommendations based on user profiles, or citation export.
- Using generated text as an authoritative academic source.
- Replacing the Go API with Supabase Data APIs, Edge Functions, Auth, Storage, or client-side database access.

## 7. Functional requirements

### Hybrid search

1. Existing `GET /api/v1/search` remains compatible with Chapter 1 parameters and response fields.
2. Relevance search supports `lexical`, `semantic`, and `hybrid` modes; `hybrid` is the default.
3. Lexical retrieval continues using PostgreSQL full-text search.
4. Semantic retrieval compares the query embedding with one stored embedding per thesis record.
5. Hybrid retrieval combines independently ranked lexical and semantic candidates using reciprocal rank fusion.
6. Existing year, division, item-type, abstract-availability, sorting, and pagination behavior remains available.
7. Exact title and author searches must not regress because semantic results were added.
8. Results without embeddings remain discoverable through lexical search.
9. Scores are useful only for ordering and must not be displayed as calibrated percentages.
10. An embedding-provider failure returns a useful error for semantic-only search and falls back to lexical results for hybrid search.

### Related theses

1. A user can request theses related to an existing record URI.
2. Related results use the selected record's stored embedding and exclude the selected record.
3. Results support an optional division filter and a limit from 1 to 20.
4. Records without an embedding return an explicit unavailable response; no relationship is invented.
5. Every related result includes its authoritative repository URI.

### Research trends

1. A user can view record counts by source year, division, item type, and normalized subject label.
2. Trend filters use stored metadata only and may be combined with year and division filters.
3. Missing values are reported separately rather than guessed.
4. The UI describes trends as corpus coverage, not the total research output of UPI.
5. Generated topic labels or inferred causal claims are not included.

### Grounded synthesis

1. Synthesis begins only after explicit user action; ordinary search never incurs generation cost.
2. The server retrieves context itself using hybrid search and sends only the top eligible records to the model.
3. Only records with available abstracts can support generated claims.
4. The response cites repository records with stable numbered markers and returns each citation's title and URI.
5. Every returned citation must reference a record included in model context.
6. Unsupported citation identifiers invalidate the generated response instead of reaching the user.
7. The answer states that it is based on metadata and available abstracts.
8. When retrieval produces insufficient evidence, the system returns an explicit insufficient-evidence response.
9. Model, timeout, token limit, latency, and usage are recorded without logging full user queries or generated answers by default.
10. If generation fails, retrieved search results remain available.

### Embedding lifecycle

1. One canonical embedding input is built from title, abstract when available, authors, subjects, and divisions.
2. The system records the input hash, model identifier, embedding dimensions, and embedding timestamp.
3. An embedding command processes records with missing or stale embeddings in bounded batches.
4. Re-running the command skips unchanged records produced by the active model.
5. A failed record reports its URI and error while preserving its searchable metadata.
6. Embeddings are updated only after a valid response with the configured dimensions.
7. Changing the model or canonical input format requires an explicit full re-embedding run.

### Corpus quality

1. The audit reports total and unique records, degree programs, yearly coverage, missing fields, duplicate URIs, probable duplicate titles, and invalid source values.
2. Searchable normalized values may be stored, but original source values remain preserved.
3. Division, item-type, and subject normalization uses reviewed deterministic mappings.
4. Probable duplicates are reported for review and are not automatically deleted or merged.
5. Adding a degree program requires representative judged queries for that program.

### Supabase deployment portability

1. Local development continues using Docker Compose PostgreSQL with pgvector.
2. Production can use Supabase as hosted PostgreSQL through the existing server-side `DATABASE_URL` path.
3. The same ordered SQL migrations create compatible schemas locally and on Supabase.
4. The Go API connects directly or through Supabase session pooling; browser code never receives a database credential.
5. Import, embedding, migration, backup, and restore operations use a direct administrative connection separate from the application's least-privilege runtime connection.
6. TLS is required for remote database traffic.
7. Supabase-specific SDKs and Data APIs are not required for search, related records, trends, or synthesis.

## 8. API contract

### Search extension

```text
GET /api/v1/search?q=pengenalan+penyakit+tanaman&mode=hybrid&page=1&limit=10
```

New optional parameter:

| Parameter | Values | Default |
| --- | --- | --- |
| `mode` | `lexical`, `semantic`, `hybrid` | `hybrid` |

Chapter 1 filters and sorts remain unchanged. `sort=relevance` uses the selected retrieval mode. `sort=title` and `sort=date` use the selected mode to determine matching candidates, then apply the requested order.

### Related theses

```text
GET /api/v1/related?uri=http%3A%2F%2Frepository.upi.edu%2Fid%2Feprint%2F123&limit=6
```

Response:

```json
{
  "source_uri": "http://repository.upi.edu/id/eprint/123",
  "results": [
    {
      "title": "Related thesis",
      "abstract": "Available abstract",
      "authors": ["Example Author"],
      "item_type": "Thesis (S1)",
      "subjects": "Q Science > QA Mathematics",
      "divisions": "Program Studi Ilmu Komputer",
      "date_deposited": "2025-01-15T03:20:00Z",
      "source_year": 2025,
      "uri": "http://repository.upi.edu/id/eprint/456",
      "score": 0.74
    }
  ]
}
```

### Trends

```text
GET /api/v1/trends?division=Program+Studi+Ilmu+Komputer
```

Response:

```json
{
  "total": 820,
  "missing_abstracts": 0,
  "by_year": [{ "value": "2026", "count": 42 }],
  "by_division": [{ "value": "Program Studi Ilmu Komputer", "count": 820 }],
  "by_item_type": [{ "value": "Thesis (S1)", "count": 818 }],
  "by_subject": [{ "value": "Q Science", "count": 120 }]
}
```

### Synthesis

```text
POST /api/v1/answer
Content-Type: application/json
```

Request:

```json
{
  "query": "Compare approaches used for plant disease detection",
  "year": 2025,
  "division": "Program Studi Ilmu Komputer"
}
```

Response:

```json
{
  "answer": "The retrieved abstracts describe two approaches ... [1][2]",
  "basis": "Generated from repository metadata and available abstracts only.",
  "citations": [
    {
      "id": 1,
      "title": "Example thesis",
      "uri": "http://repository.upi.edu/id/eprint/123"
    }
  ],
  "insufficient_evidence": false
}
```

The trimmed synthesis query must contain 1–500 characters. Retrieval filters follow search validation. The server chooses the context count within its fixed token budget; clients cannot submit arbitrary source text.

## 9. UX requirements

- Search remains useful without requesting synthesis.
- Search, related results, trends, and synthesis have separate loading, empty, and error states.
- The default search experience is hybrid; comparison modes may be exposed as an evaluation control without technical jargon in the primary UI.
- Each result can open its source and request related theses.
- Synthesis uses an explicit action such as `Summarize retrieved abstracts`.
- Citations are keyboard-accessible links to repository URIs.
- Generated text is visually labeled and never mixed into repository metadata fields.
- Insufficient evidence, missing abstracts, provider failures, and partial embedding coverage are visible.
- Trend charts include text equivalents and describe corpus coverage accurately.
- Existing semantic HTML, visible focus, touch targets, and reduced-motion behavior remain intact.

## 10. Success criteria

Before release:

- Audit at least 2,000 valid unique records and record actual program and field coverage.
- Build at least 100 manually judged queries, covering every included degree program and lexical, conceptual, bilingual, abbreviation, exact-title, and exact-author intents.
- Hybrid retrieval improves `nDCG@10` by at least 10% relative to the Chapter 2 lexical baseline on the conceptual and bilingual subset.
- Hybrid retrieval does not reduce exact-title or exact-author success: expected records remain first.
- At least 85% of the full evaluation set places a judged relevant result in the top 5.
- Every valid unique URI remains searchable by exact title.
- At least 30 synthesis prompts are manually evaluated; at least 90% of factual claims are supported by a cited retrieved abstract.
- Every generated citation resolves to a retrieved repository record; invalid citation identifiers never reach the client.
- Hybrid search p95 remains below 500 ms locally for the Chapter 2 corpus, measured separately from external model latency.
- Embedding reruns skip unchanged records and report all failures.
- A clean Supabase database can apply all migrations, import the corpus, store vectors, serve the Go API, and pass search and readiness smoke checks.
- The Supabase runtime connection uses a least-privilege database role and no database secret is exposed to the frontend.
- Retrieval, synthesis validation, frontend, corpus audit, and migration checks pass through documented commands.

## 11. Chapter 2 exit gate

Chapter 2 is complete when the expanded corpus is audited, hybrid retrieval measurably beats the lexical baseline on queries designed to expose lexical gaps, exact lookup remains reliable, related and trend exploration work, generated synthesis is grounded in cited available abstracts, and the same application passes its production smoke checks against Supabase PostgreSQL.

No feature should be credited for improvement without comparison against the Chapter 1 retrieval path.

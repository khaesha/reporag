# Chapter 1 PRD: Thesis Metadata Discovery

Status: Draft 1  
Owner: SearchLens team  
Scope: Chapter 1 only

## 1. Product summary

Chapter 1 turns UPI repository metadata from multiple degree programs into a searchable thesis catalog. A user enters keywords, optionally narrows the results, and opens the original repository record through its URI.

This release is a retrieval product, not a question-answering system. It searches only the metadata and abstracts that are legally available. It must never imply access to a thesis PDF or unsupported chapter-level content.

## 2. Corpus baseline

The checked-in starter corpus currently contains Computer Science records:

- 820 unique records from 18 yearly JSON files (2009–2026).
- 820 titles and 820 repository URIs.
- 163 abstracts; 657 records (80%) have no abstract.
- 818 S1 theses and 2 S2 theses.
- Authors, subjects, divisions, depositor, and deposit date where supplied by the source.

The source filename supplies `source_year`. It is not inferred from `date_deposited`, because deposit year and repository listing year can differ.

This is a starting dataset, not the intended corpus boundary. Additional degree programs may be added later when they follow the same record shape. Their program identity is retained in `divisions`; no Computer Science-specific field or table is introduced.

## 3. Problem

Repository visitors need a quick way to discover relevant theses across years and degree programs. The available starter data is split across JSON files, and the current frontend displays hard-coded results. Full thesis documents are restricted, so discovery must work well from titles and metadata alone.

## 4. Users and primary jobs

### Student or researcher

- Find theses related to a topic or author.
- Narrow results by repository year, division, or item type.
- Understand which records include an abstract.
- Open the authoritative repository page.

### Maintainer

- Import the supplied JSON files repeatedly without creating duplicates.
- Add files from another degree program without changing backend code when they use the same schema.
- See invalid records instead of silently losing them.
- Run the service locally with minimal setup.

## 5. Goals

- Replace the frontend's hard-coded search results with real API results.
- Search title, abstract, authors, subjects, and divisions.
- Support relevance, title, and deposit-date sorting.
- Preserve a direct link to the repository source.
- Make missing abstracts visible.
- Establish a measurable retrieval baseline before semantic search or an LLM is added.

## 6. Non-goals

- Downloading, parsing, indexing, or summarizing restricted PDFs.
- Generating answers with an LLM.
- Vector embeddings, reranking, or a separate vector database.
- User accounts, saved searches, admin UI, or analytics dashboards.
- Rescraping the upstream repository.

## 7. Functional requirements

### Search

1. A user can submit a non-empty query.
2. Search matches titles, abstracts, authors, subjects, and divisions.
3. Title matches rank above matches in other fields.
4. Search supports filters for `source_year`, `division`, `item_type`, and abstract availability.
5. Search supports `relevance`, `title`, and `date` sorting.
6. Results are paginated with a default of 10 and a maximum of 50 per page.
7. Every result includes its repository URI.
8. A record without an abstract is labeled `Abstract unavailable`; no text is invented.
9. A search with no matches returns an empty result list, not an error.

### Import

1. A command recursively imports every `repository_*_data.json` file from a configured corpus directory.
2. `uri` is the stable unique identifier; rerunning the importer updates the existing record.
3. The importer derives `source_year` from the filename.
4. Degree program is read from `divisions`; the importer does not assume a specific program.
5. Missing optional fields are stored as null or empty arrays as appropriate.
6. A malformed record is reported with its filename and array position.
7. Import is transactional: a failed file does not leave a partial file import.
8. The command prints inserted, updated, rejected, and total counts.

### Operations

1. `GET /healthz` reports process health.
2. `GET /readyz` verifies database connectivity.
3. Requests have a configurable timeout and graceful shutdown.
4. Logs are structured and do not contain full search queries by default.

## 8. API contract

### Search request

`GET /api/v1/search?q=machine+learning&year=2024&page=1&limit=10&sort=relevance`

Optional parameters:

| Parameter | Values |
| --- | --- |
| `year` | 2009–2026 |
| `division` | Exact division value |
| `item_type` | Exact item type value |
| `has_abstract` | `true` or `false` |
| `sort` | `relevance`, `title`, or `date` |
| `page` | Positive integer |
| `limit` | 1–50 |

### Search response

```json
{
  "query": "machine learning",
  "page": 1,
  "limit": 10,
  "total": 42,
  "results": [
    {
      "title": "Example thesis",
      "abstract": null,
      "authors": ["Example Author"],
      "item_type": "Thesis (S1)",
      "subjects": "Q Science > QA Mathematics",
      "divisions": "Program Studi Ilmu Komputer",
      "date_deposited": "2024-01-15T03:20:00Z",
      "source_year": 2024,
      "uri": "http://repository.upi.edu/id/eprint/123",
      "score": 0.81
    }
  ]
}
```

The score is meaningful only within the current result set and must not be displayed as a calibrated percentage match.

## 9. UX requirements

- Search begins only after explicit submission.
- Loading, empty, API-error, and results states are distinct.
- Result cards show title, authors, year, item type, abstract availability, and source link.
- Filters remain selected while moving between result pages.
- The interface says `Search thesis metadata and available abstracts`.
- Remove claims such as `multimodal`, `enterprise-grade`, `verified index`, and `generate answers` in Chapter 1.

## 10. Success criteria

Before release, create a fixed evaluation set of at least 30 real queries:

- At least 24/30 queries place a manually judged relevant result in the top 5.
- Exact title and author queries place the expected record first.
- Every valid unique URI in the configured corpus is importable and searchable by exact title.
- Reimporting the same files leaves the record count unchanged.
- Search API p95 is below 300 ms locally for this corpus.
- API and importer checks pass in one documented command each.

## 11. Chapter 1 exit gate

Chapter 1 is complete when the available corpus is imported, the frontend performs real searches across its degree programs, the evaluation target is met, and the known corpus limitations are visible to users. New same-schema data can continue to be imported after this gate. Only then should Chapter 2 add embeddings and LLM-generated synthesis.

# RAG Glossary

This guide explains the terms used in SearchLens. Definitions favor intuition over formal mathematics, then connect each term to this repository.

## RAG in one minute

**RAG (Retrieval-Augmented Generation)** answers a question in two main steps:

1. **Retrieve:** search a trusted collection and select relevant records.
2. **Generate:** give those records to a language model so it can write an answer grounded in them.

```text
User question
    ↓
Search the corpus
    ↓
Rank and select relevant records
    ↓
Give selected evidence to the language model
    ↓
Return an answer with source citations
```

SearchLens can stop after retrieval and show search results. Its optional synthesis feature performs the generation step. It uses only repository metadata and available abstracts; it does not have thesis PDFs or chapter text.

## Core data terms

### Corpus

The complete collection that the system can search. SearchLens's corpus is the imported UPI thesis metadata and available abstracts. If information is not in this corpus, the system should not claim that it knows it from the repository.

### Document or record

One searchable item in the corpus. In SearchLens, one document is one thesis record identified by its repository URI. It contains fields such as title, authors, year, degree program, subjects, and possibly an abstract.

### Metadata

Information describing a document rather than its full content. Title, author, year, subject, degree program, and URI are metadata.

### Abstract

A short summary supplied by the source repository. SearchLens may use an available abstract as evidence. A missing abstract stays missing; the application never invents one.

### Document chunk

A smaller piece cut from a long document, often a paragraph or section. Chunking lets retrieval find the relevant part instead of comparing a query with an entire long document.

SearchLens does **not** chunk documents today. Each thesis record has one embedding built from bounded metadata and its available abstract. Chunking becomes useful only if legally available content becomes much longer than abstracts.

### Query

The words a user submits to search. For semantic or hybrid search, SearchLens also converts the query into an embedding.

## Retrieval terms

### Retrieval

The process of finding records that may answer the query. Retrieval chooses the evidence before any generated answer is written. Strong generation cannot repair missing or irrelevant evidence reliably.

### Lexical search

Search based on words that appear in both the query and document. It is strong for exact titles, author names, abbreviations, and distinctive terms.

Example: a search for `backpropagation rainfall` favors records containing those words. A record using different words for the same idea may be missed.

### Semantic search

Search based on meaning rather than exact word overlap. A model converts the query and documents into vectors, then the database finds vectors that are close together.

Example: `predicting rain with neural networks` may find a relevant abstract that says `backpropagation for rainfall forecasting`, even though the wording differs.

### Hybrid search

Search that combines lexical and semantic rankings. Lexical search protects exact matches; semantic search helps with paraphrases, concepts, and bilingual wording.

SearchLens combines the two ranked lists using reciprocal rank fusion instead of treating their raw scores as directly comparable.

### Reciprocal Rank Fusion (RRF)

A simple method for merging ranked lists. A result earns points based on its position in each list; appearing near the top of both lists gives it more combined weight. RRF uses rank positions, so lexical and vector scores do not need the same scale.

### Embedding

A list of numbers produced by a model to represent meaning. Similar text should produce embeddings that are close in vector space.

SearchLens stores one 1,536-dimension embedding per thesis record and creates a temporary embedding for each semantic or hybrid query.

### Vector

The numeric list that holds an embedding. A vector is useful to the system even though individual numbers are not meaningful to a person.

### Dimension

The number of values in a vector. SearchLens uses 1,536 dimensions. Query and document vectors must use the same model and dimension count before they can be compared safely.

### Similarity and distance

Ways to measure how close two vectors are. Higher cosine similarity means more alike; lower cosine distance means more alike. They describe the same relationship from opposite directions.

### Query embedding and document embedding

A **document embedding** represents a stored corpus record and can be reused across searches. A **query embedding** represents one user's current query and is created at search time.

### pgvector

A PostgreSQL extension that stores vectors and compares them inside the existing database. SearchLens uses it instead of adding a separate vector database.

### Exact vector search

Comparison against every eligible stored vector to find the nearest results. It gives exact nearest neighbors and is fast enough for the current SearchLens corpus.

### Approximate vector search and HNSW

Approximate search uses an index to search large vector collections faster, with a small chance of missing a true nearest result. HNSW is a common approximate index. SearchLens does not need it yet because measured exact search is already below the latency target.

### Reranking

A second ranking pass over a small set of retrieved candidates, usually with a more expensive model. It may improve ordering, but adds cost and latency. SearchLens has no measured need for it yet.

### Filter

A strict condition such as year, degree program, item type, or abstract availability. Filtering removes records that do not meet the condition; it is different from ranking the remaining records by relevance.

### Top-k

The first `k` results after ranking. `Top-5` means the first five results; `@10` means a metric is calculated using at most the first ten.

## Evaluation terms

Retrieval evaluation starts with a fixed set of queries and human judgments about which corpus records are relevant. SearchLens compares returned rankings with those judgments.

Assume one query has four known relevant records. Search returns three of them within its first ten results, and the first relevant result appears at rank 2. This example helps distinguish the metrics below.

### Relevance judgment

A human label saying whether a record is useful for a query, sometimes with grades such as highly relevant, partly relevant, or not relevant. Metrics need these judgments; they cannot determine relevance from the search system's own score.

### Recall@10

The share of all known relevant records found in the first ten results.

In the example, three of four relevant records appear in the top ten, so `Recall@10 = 3 / 4 = 0.75`. Recall rewards finding all relevant material, but does not care much whether it appears at rank 1 or rank 10.

### MRR@10

**Mean Reciprocal Rank at 10** measures how quickly the first relevant result appears. For one query, use `1 / rank` of the first relevant result; use zero if none appears in the top ten. Then average across all queries.

In the example, the first relevant result is rank 2, so its reciprocal rank is `1 / 2 = 0.5`. Rank 1 scores 1.0, rank 5 scores 0.2. MRR focuses only on the first relevant result.

### nDCG@10

**Normalized Discounted Cumulative Gain at 10** rewards highly relevant results near the top. Results farther down receive less credit. The score is divided by the best possible ordering, producing a value from 0 to 1.

A score near 1 means the ranking is close to ideal. Unlike MRR, nDCG can use several results and graded relevance, so putting a highly relevant record at rank 1 matters more than putting it at rank 10.

### Top-5 success

The share of queries with at least one relevant result in the first five positions. Each query is a simple yes or no.

The example is a success because its first relevant result is rank 2. This metric answers, "Will a user probably see something useful without looking far?" It does not measure how many relevant records were found.

### Baseline

A known result used for comparison. SearchLens uses lexical retrieval as a baseline when measuring whether semantic or hybrid retrieval improves conceptual and bilingual searches.

## Latency terms

### Latency

How long an operation takes. SearchLens separates database retrieval time from external model time so a slow provider is not mistaken for a slow database query.

### p50

The median latency. Half of measured requests finish at or below this value and half take longer.

### p95

The 95th-percentile latency. About 95 of 100 measured requests finish at or below this value; about five are slower. It describes a slower but common user experience better than an average alone.

### Local retrieval p95

The p95 time spent retrieving results from the local database. It excludes query-embedding provider time. SearchLens's Chapter 3 local retrieval p95 is below 500 ms for lexical, semantic, hybrid, related, and trend workloads.

### Provider latency

Time spent waiting for an external model provider, such as creating a query embedding or generating an answer. Network and model processing make this separate from local retrieval.

### End-to-end latency

Total time observed for the request, including local work, provider calls, and other application overhead. For semantic and hybrid search, this is normally higher than local retrieval latency.

## Generation and trust terms

### Generation

Writing a natural-language answer with a language model. In RAG, generation happens after retrieval and receives selected records as context.

### Context

The evidence and instructions sent to the language model. SearchLens context contains selected metadata and available abstracts, with size limits.

### Token

A small unit of text processed by a model. Tokens are not exactly words. Input and output token counts affect model limits, latency, and cost.

### Grounded answer or grounded synthesis

An answer whose factual claims are supported by retrieved evidence. SearchLens asks the model to cite corpus records and validates that returned citation identifiers belong to the supplied evidence.

### Citation

A link from a generated claim back to its source record. SearchLens preserves the repository URI as the authoritative destination. A valid citation proves which record was referenced; human review is still needed to confirm that the record supports the claim.

### Hallucination

A confident-looking statement that is unsupported or false. RAG reduces this risk by supplying evidence, but does not remove it. Retrieval quality, prompt rules, citation validation, and human evaluation all matter.

### Insufficient evidence

The correct response when retrieved records cannot support an answer. Missing abstracts are normal in this corpus. SearchLens should report insufficient evidence instead of filling the gap with invented text.

## Data lifecycle terms

### Import

Reading source corpus files, validating records, and storing them in the database. SearchLens uses the repository URI as each record's stable unique key.

### Idempotent operation

An operation that can be repeated without changing the final logical result after the first successful run. For embeddings, an unchanged rerun skips all current vectors. True no-op import behavior is scheduled as the first Chapter 4 task.

### Migration

A versioned database change, such as creating a table or adding vector columns. Migrations let an empty database reach the expected schema in a repeatable order.

### Index

An extra database structure that speeds up certain lookups at the cost of storage and update work. An index should be added for a measured need, not merely because the database supports it.

## Reading SearchLens metrics correctly

- High retrieval metrics mean the judged queries usually return relevant records. They do not prove every possible query works.
- Low local retrieval p95 means the database is fast on this corpus and machine. It does not include all provider or network delay.
- Valid citations mean cited IDs map to retrieved corpus records. They do not automatically prove every generated claim is supported.
- Missing abstracts limit what RAG can answer. The safe behavior is an honest limitation or insufficient-evidence response.
- Current results describe 2,190 indexed records across three degree programs. Larger or different corpora need new evaluation.

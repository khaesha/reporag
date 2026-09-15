# Chapter 2 Phase 0 Evidence

Status: Complete  
Measured: 2026-09-15

## Corpus

The recovered source contains 42 JSON files, 2,191 rows, and 2,190 unique URIs. Three anomalous URIs are quarantined without changing their raw records, leaving 2,187 valid unique records:

- Program Studi Pendidikan Ilmu Komputer: 955
- Program Studi Ilmu Komputer: 819
- Program Studi Fisika - S1: 413

Three abstracts and five subject values are missing. Field types and deposit dates contain no invalid values. One duplicate URI and 34 probable duplicate-title groups require review rather than automatic deletion. Nine distinct subject strings cannot be mapped deterministically and remain explicitly unmapped. Full evidence and corpus digest are in `corpus-audit.json`.

## Evaluation

`evaluation.json` contains 100 draft queries: 15 exact-title, 15 exact-author, 20 lexical, 20 conceptual, 15 bilingual, and 15 abbreviation queries. All three accepted degree programs and multiple relevant URI judgments are represented. Relevance uses grades 1–3; grades 2 and 3 count as relevant for Recall, MRR, and top-5 success. nDCG uses the full grades.

Normalization mappings and the 100-query judgment set were approved on 2026-09-15.

The Chapter 1 lexical baseline is:

| Metric | Result |
| --- | ---: |
| Recall@10 | 0.6100 |
| MRR@10 | 0.5950 |
| nDCG@10 | 0.5989 |
| Top-5 success | 0.6100 |
| Local p95 | 1.24 ms |

Conceptual nDCG@10 is 0 and bilingual nDCG@10 is 0.2, establishing the intended semantic-retrieval gap. Exact-title nDCG@10 remains 1.0.

## Models and limits

OpenRouter is the single API provider and uses `OPENROUTER_API_KEY`. The benchmark compares:

- `openai/text-embedding-3-small`, 1,536 dimensions, recorded list price $0.02 per million input tokens.
- `openai/text-embedding-3-large`, 1,536 dimensions, recorded list price $0.13 per million input tokens.
- `google/gemini-embedding-2`, 1,536 dimensions, recorded list price $0.20 per million input tokens.

Full-corpus benchmark result: `google/gemini-embedding-2` is selected. It reached 0.9040 conceptual-plus-bilingual nDCG@10, compared with 0.8286 for `text-embedding-3-large` and 0.6891 for the lower-cost `text-embedding-3-small`. All beat the 0.0857 lexical subset baseline; Google was selected for its material quality advantage. Its measured full-corpus input cost was $0.212592, versus $0.177709 and $0.027340. Full details are in `embedding-benchmark.json`.

Generation is locked to `openai/gpt-5.6-luna`: 16,000 input tokens, 800 output tokens, 30-second timeout, and four concurrent requests. At the recorded $0.20/$1.20 per-million input/output prices, the configured maximum is approximately $0.00416 per answer. Generated output is based on repository metadata and available abstracts only.

`text-embedding-3-small` is viable budget fallback, but is not active because its semantic relevance was materially lower. All generated output remains based on repository metadata and available abstracts only.

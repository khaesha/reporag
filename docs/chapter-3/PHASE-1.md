# Chapter 3 Phase 1 Evidence

Status: Complete
Measured: 2026-10-08

## Exact-title verification

The Chapter 2 verifier reported 93 failures. All 93 have unique normalized titles between 201 and 296 characters, so the API rejected their full-title requests at the former 200-character query limit. This was a shared validation defect, not a retrieval, source, or duplicate-title defect.

Search now accepts up to 500 characters, matching the longest corpus title while retaining a bounded input. The verifier groups normalized duplicate titles, keeps a 10-result page size, and follows deterministic pagination until every URI in the group is found. The final result records a pass disposition, page, and rank for every valid URI:

| Item | Result |
| --- | ---: |
| Valid URIs checked | 2,187 |
| Normalized title groups | 2,152 |
| Passed URIs | 2,187 |
| Failures or approved limitations | 0 |

Full output: [`exact-titles.json`](exact-titles.json).

## Retrieval and latency

The unchanged 100-query set preserved lexical, semantic, and hybrid quality. Hybrid Recall@10/MRR@10/nDCG@10/top-5 success remains 0.9767/0.9503/0.9503/0.9900; exact-title and exact-author judged queries remain 1.0.

| Workload | Local retrieval p95 | Provider p95 |
| --- | ---: | ---: |
| Lexical evaluation | 2.41 ms | 0 ms |
| Semantic evaluation | 54.56 ms | 1,311.21 ms |
| Hybrid evaluation | 58.15 ms | 1,358.79 ms |
| Related records | 21.56 ms | N/A |
| Trends | 3.95 ms | N/A |

All local retrieval p95 values remain below 500 ms. Provider time is reported separately and is not counted as database retrieval time. Semantic retrieval timing is the current exact pgvector candidate-search path; no approximate index is justified.

Full outputs: [`lexical-evaluation.json`](lexical-evaluation.json), [`semantic-evaluation.json`](semantic-evaluation.json), [`hybrid-evaluation.json`](hybrid-evaluation.json), and [`related-trend-latency.json`](related-trend-latency.json).

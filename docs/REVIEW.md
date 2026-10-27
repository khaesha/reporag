# Chapter 3 Review and Chapter 4 Handoff

Status: accepted local release; Chapter 4 may begin under the recorded follow-up decisions  
Reviewed: 2026-10-19  
Reviewed range: Chapter 3 plan commit `fccda35` through Phase 4 merge commit `08f99b9`

## Executive summary

Chapter 3 delivered the intended local-release work. Exact-title retrieval now covers every valid URI, Chapter 2 retrieval quality remains above target, synthesis has a 30-prompt claim review, and the database was rebuilt, backed up, restored, and exercised through all API surfaces.

The user accepted the Chapter 3 evidence on 2026-10-19. Repeat-import cleanup moves to the first Chapter 4 delivery unit, browser-state verification remains manual tech debt, and the synthesis review is complete for the current dataset, including its known missing-abstract limits.

The release claim remains local-only. Nothing in Chapter 3 proves public or hosted production readiness.

## Repository state at review

- Local and remote `main` both end at `08f99b9` (`Complete Chapter 3 Phase 4 (#21)`).
- Phase 4 commit `4eabc4f` is contained in `main` through that merge.
- Existing `.DS_Store` changes and untracked design/backup files were left untouched.

The Chapter 3 acceptance and merge prerequisites for Chapter 4 are satisfied.

## Delivered

| Area | Result |
| --- | --- |
| Release boundary | Reproducible one-host release with PostgreSQL restricted to loopback; no public-production claim |
| Corpus | 42 files, 2,191 source rows, 2,190 unique URIs, 2,187 valid non-quarantined URIs across three degree programs |
| Exact titles | Query limit raised from 200 to 500 characters; all 2,187 valid URIs pass; highest recorded rank is 3 |
| Retrieval | Lexical, semantic, and hybrid evaluation rerun on the unchanged 100-query set; related and trend latency added |
| Synthesis | 30 prompts, including six intended insufficient-evidence cases; four returned insufficient evidence and two returned grounded alternatives; 85 claims judged supported |
| Operations | Empty-database migration, import, embedding, custom-format backup, separate restore, count verification, and restored API smoke |
| Release smoke | Readiness, lexical, semantic, hybrid, related, trends, and synthesis paths recorded against the active local API |
| Documentation | Local deployment, backup, restore, rollback, release checks, and local-only limits documented |

## Quality evidence

### Retrieval

| Mode | Recall@10 | MRR@10 | nDCG@10 | Top-5 success | Local retrieval p95 |
| --- | ---: | ---: | ---: | ---: | ---: |
| Lexical | 0.6100 | 0.5950 | 0.5989 | 0.6100 | 2.41 ms |
| Semantic | 0.9767 | 0.9198 | 0.9274 | 0.9900 | 54.56 ms |
| Hybrid | 0.9767 | 0.9503 | 0.9503 | 0.9900 | 58.15 ms |

Related-record p95 is 21.56 ms and trend p95 is 3.95 ms. All local retrieval measurements remain below the 500 ms gate. Provider time is recorded separately.

### Synthesis

| Measure | Result |
| --- | ---: |
| Prompts | 30 |
| Successful responses | 30 |
| Factual claims judged | 85 |
| Claims marked supported | 85 (100%) |
| Citation URI/title mappings | 30 of 30 valid |
| Input / output tokens | 104,261 / 6,429 |
| Estimated workload cost | $0.028567 |
| End-to-end p95 | 5.41 s |

Generation used `openai/gpt-5.6-luna`. Search and synthesis remain limited to repository metadata and available abstracts.

### Database operations

- PostgreSQL 18.6 and pgvector 0.8.6.
- Clean build produced 2,190 documents and 2,190 embeddings.
- Backup checksum: `1500f08168f17f2fa42b14e30c501e2ca70673cf6436879a2aa81e81190516d8`.
- Separate restore preserved 2,190 documents and 2,190 embeddings.
- Restored readiness, lexical, semantic, hybrid, related, trends, and synthesis requests passed.
- Generated disposable databases were removed; recovery still depends on retaining the external dump.

## Review decisions

### Accepted on 2026-10-19

1. **Phase 4 merge confirmed.** Local and remote `main` contain Phase 4 through merge commit `08f99b9`.
2. **Current database evidence accepted without another import.** The recorded database contains the expected 2,190 documents and embeddings. The second import's 2,191 updates are accepted for Chapter 3; making unchanged imports true no-ops is the first Chapter 4 task before corpus expansion.
3. **Browser verification deferred.** Existing keyboard search, related, synthesis, and trends checks are accepted. Remaining loading, empty, failure/retry, focus, touch, and insufficient-evidence checks stay as manual tech debt and do not block Chapter 4.
4. **Synthesis review accepted for the current dataset.** The 30-prompt review, 85 supported claims, and valid citation mappings close the Chapter 3 gate. Missing abstracts are source-data limitations and do not require invented support or regenerated content.

Details and closure rules are tracked in [TECH_DEBT.md](TECH_DEBT.md).

## Checks rerun on 2026-10-19

| Check | Result |
| --- | --- |
| Script tests | Passed: 22 |
| Frontend tests | Passed: 4 |
| Frontend lint | Passed |
| TypeScript | Passed |
| Next.js production build | Passed |
| Backend vet and tests without database integration | Passed |
| Backend database integration rerun | Not reproduced: local PostgreSQL at `127.0.0.1:55432` was not running |

The committed Phase 4 evidence records a successful database-enabled backend run. This review verified the code-level suites but did not recreate the external model workload, restored database, live API smoke, or browser session.

## Chapter 4 decision

Chapter 4 may begin. Before implementation:

- write and approve the Chapter 4 PRD, architecture, and phased plan;
- make true no-op repeat imports the first delivery unit before adding corpus data; and
- approve corpus sources and degree-program order.

Once those conditions are met, Chapter 4 should remain focused on corpus expansion and incremental refresh. HNSW, reranking, chunking, background workers, separate vector infrastructure, provider abstraction, accounts, and synchronization still lack measured need.

## Known product limits

- Three anomalous URIs remain quarantined without modifying source files.
- One duplicate URI and 34 probable duplicate-title groups still require human data review; no automatic merge or deletion is approved.
- Three abstracts and five subject values are missing; nine subject strings remain unmapped.
- Restricted PDFs and chapter-level content remain unavailable and must never be implied.
- Hosted deployment remains [To Be Discussed](TECH_DEBT.md).

## Source evidence

- [RAG glossary](GLOSSARY.md)
- [Chapter 3 PRD](chapter-3/PRD.md)
- [Chapter 3 architecture](chapter-3/ARCHITECTURE.md)
- [Chapter 3 plan](chapter-3/PLAN.md)
- [Phase 1 retrieval evidence](chapter-3/PHASE-1.md)
- [Synthesis results](chapter-3/synthesis-results.json)
- [Synthesis claim review](chapter-3/synthesis-review.md)
- [Database operations evidence](chapter-3/database-operations.json)
- [Phase 4 release evidence](chapter-3/PHASE-4.md)

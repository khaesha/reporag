# Chapter 3 Phase 4: Local release gate

Status: complete on 2026-10-09. This is a local, one-host release result only.
Hosted deployment remains [To Be Discussed](../TECH_DEBT.md).

## Final evidence

- Environment: PostgreSQL 18.6, pgvector 0.8.6, 2,190 documents, and 2,190 embeddings (`rtk make migrate-report`).
- Corpus: 42 files, 2,191 source rows, 2,190 unique URIs, 2,187 valid unique URIs; SHA-256 `0162b461c5c8af7d199dd6adc9eb998143a9c2d1063aac7d06c08fbff8057e0d` in [final-corpus-audit.json](final-corpus-audit.json).
- Retrieval: the unchanged 100-query evidence retains hybrid Recall@10/MRR@10/nDCG@10/top-5 success of 0.9767/0.9503/0.9503/0.9900; exact-title and exact-author queries remain 1.0 ([Phase 1](PHASE-1.md)). Local retrieval p95 stays below 500 ms.
- Synthesis: the 30-prompt review has 100% supported claim judgments and valid citations; it used `openai/gpt-5.6-luna`, 104,261 input tokens, 6,429 output tokens, estimated cost $0.028567, and 5.41 s p95 ([synthesis review](synthesis-review.md)).
- Backup/restore: [database-operations.json](database-operations.json) records the Phase 3 clean build, idempotent re-import/re-embed, restored 2,190/2,190 counts, all seven restored API checks, and dump SHA-256 `1500f08168f17f2fa42b14e30c501e2ca70673cf6436879a2aa81e81190516d8`. The generated databases were removed. Recovery requires retaining an external `BACKUP_DIR` dump, never the active database.
- Live API: [release-smoke.json](release-smoke.json) records readiness, lexical, semantic, hybrid, related, trends, and synthesis success against the active local connection.

## Checks run

| Gate | Result |
| --- | --- |
| `rtk env ... make backend-test` | passed with database integration configuration |
| `rtk make migrate-report` | PostgreSQL 18.6 / pgvector 0.8.6 / 2,190 documents / 2,190 embeddings |
| `rtk node scripts/audit-corpus.mjs docs/repository-data docs/chapter-3/final-corpus-audit.json` | passed |
| `rtk node --test scripts/*.test.mjs` | passed: 22 tests |
| `rtk node scripts/release-smoke.mjs http://localhost:8080 docs/chapter-3/release-smoke.json` | passed: seven API paths |
| `rtk npm test`, `rtk npm run lint`, `rtk npx tsc --noEmit`, `rtk npm exec -- next build --webpack` from `apps/frontend` | passed |

## Browser release check

Chrome against `http://localhost:3000` confirmed keyboard search for
`backpropagation` (51 results), related theses, synthesis loading, completed
synthesis with repository citations, and corpus trends (2,190 indexed records).
The lexical empty-result boundary was also checked at the API. Static inspection
of the rendered client states confirms empty/error retry controls,
`focus-visible` outlines, and `min-h-11` (44px) touch targets; no public browser
or hosted service was used. The browser switched to an existing user tab before
a second manual pass, so no user tab was altered.

No Chapter 3 code changes occurred after the Phase 3 restore verification; the
recorded restore smoke is the applicable restored-database operation check.

## Reproduce

Start the local database, API, and frontend using [DEPLOYMENT.md](DEPLOYMENT.md),
then write new output outside the repository:

```sh
rtk node scripts/release-smoke.mjs http://localhost:8080 /absolute/path/release-smoke.json
rtk make phase3-verify BACKUP_DIR=/absolute/path/outside/repository
```

The second command uses only generated disposable loopback databases and leaves
the active database out of scope.

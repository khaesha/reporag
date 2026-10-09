# Chapter 3 Phase 3 Evidence

Status: Complete
Measured: 2026-10-09

## Isolated lifecycle

`rtk make phase3-verify BACKUP_DIR=/private/tmp/searchlens-phase3-backups.r7eNvn` created uniquely named build and restore databases on the loopback PostgreSQL server. The active `searchlens` database was not a target. Both generated databases were removed after the run; the temporary external verification dump remains outside the repository. Operators must use a retained external `BACKUP_DIR` for recovery use.

## Results

| Check | Result |
| --- | --- |
| PostgreSQL / pgvector | 18.6 / 0.8.6 |
| First import | 2,190 inserted, 1 updated, 0 rejected; 2,191 source rows |
| Second import | 0 inserted, 2,191 updated, 0 rejected; no duplicate documents |
| First embedding | 2,190 embedded, 0 skipped, 0 failed |
| Second embedding | 0 embedded, 2,190 skipped, 0 failed |
| Source and restored counts | 2,190 documents and 2,190 embeddings |
| Backup | Custom format; SHA-256 `1500f08168f17f2fa42b14e30c501e2ca70673cf6436879a2aa81e81190516d8` |
| Restored smoke checks | readiness, lexical, semantic, hybrid, related, trends, synthesis passed |

Machine-readable evidence is in [`database-operations.json`](database-operations.json). The operational command and cleanup/rollback constraints are in [`DEPLOYMENT.md`](DEPLOYMENT.md).

# Chapter 3 Phase 0 Evidence

Status: Complete
Measured: 2026-10-08

## Local release environment

Chapter 3 targets a reproducible local release on one host. PostgreSQL remains bound to loopback through Docker Compose, and the application continues using the existing server-side `DATABASE_URL` contract.

The measured local database baseline is:

| Item | Result |
| --- | --- |
| PostgreSQL | 18.6 (Debian 18.6-1.pgdg13+2) |
| pgvector | 0.8.6 |
| Documents | 2,190 |
| Current embeddings | 2,190 |

Hosted deployment is **To Be Discussed**. This evidence does not support a public-production claim.

## Corpus

The unchanged corpus contains 42 JSON files, 2,191 source rows, 2,190 unique URIs, and 2,187 valid non-quarantined URIs. Its SHA-256 digest is `0162b461c5c8af7d199dd6adc9eb998143a9c2d1063aac7d06c08fbff8057e0d`.

Full Chapter 3 corpus evidence is recorded in [`corpus-audit.json`](corpus-audit.json). Chapter 2 evidence remains immutable and historical; Chapter 3 results use new files under `docs/chapter-3`.

## Models and limits

OpenRouter remains the single model provider. Active server-side settings are:

| Purpose | Model | Limits |
| --- | --- | --- |
| Embedding | `google/gemini-embedding-2` | 1,536 dimensions; batch size 32 |
| Generation | `openai/gpt-5.6-luna` | 16,000-byte input limit; 800 output tokens; 30-second provider timeout; concurrency four |

No model identifier, application behavior, API contract, schema, or dependency changed in this phase.

## Reproduction

```sh
rtk make db-up
rtk make migrate
rtk make migrate-report
rtk node scripts/audit-corpus.mjs docs/repository-data docs/chapter-3/corpus-audit.json
rtk node --test scripts/*.test.mjs
```

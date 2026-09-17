# Chapter 2 Tech Debt

## Production hardening

- Use direct IPv6 or Supabase IPv4 add-on for administrative migration, import, embedding, backup, and restore proof.
- Replace temporary administrative API credential with `searchlens_runtime` Session Pooler credential; prove it cannot write `documents` or alter schema.
- Validate `pg_dump` and restore against a disposable Supabase project before public deployment.

These items block public production deployment. They do not block the Phase 6 functional release candidate.

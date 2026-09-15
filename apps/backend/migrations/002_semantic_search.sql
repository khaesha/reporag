BEGIN;

CREATE SCHEMA IF NOT EXISTS extensions;
CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA extensions;

ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS embedding extensions.vector(1536),
    ADD COLUMN IF NOT EXISTS embedding_model text,
    ADD COLUMN IF NOT EXISTS embedding_input_hash text,
    ADD COLUMN IF NOT EXISTS embedding_dimensions smallint,
    ADD COLUMN IF NOT EXISTS embedded_at timestamptz;

COMMIT;

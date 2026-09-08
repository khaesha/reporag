BEGIN;

CREATE TABLE documents (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uri             text NOT NULL UNIQUE,
    source_year     smallint NOT NULL CHECK (source_year BETWEEN 1900 AND 2100),
    title           text NOT NULL,
    abstract        text,
    authors         text[] NOT NULL DEFAULT '{}',
    item_type       text,
    subjects        text,
    divisions       text,
    depositing_user text,
    date_deposited  timestamptz,
    search_text     text NOT NULL,
    search_vector   tsvector GENERATED ALWAYS AS
                    (to_tsvector('simple', search_text)) STORED,
    imported_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX documents_search_idx ON documents USING gin (search_vector);
CREATE INDEX documents_year_idx ON documents (source_year);
CREATE INDEX documents_item_type_idx ON documents (item_type);

COMMIT;

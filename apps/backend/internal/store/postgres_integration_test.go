package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khaesha/reporag/apps/backend/internal/records"
)

func TestImportFileIsIdempotentAndTransactional(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	_, err = pool.Exec(ctx, `CREATE TEMP TABLE documents (
		uri text PRIMARY KEY, source_year smallint CHECK (source_year BETWEEN 1900 AND 2100),
		title text NOT NULL, abstract text, authors text[] NOT NULL, item_type text,
		subjects text, divisions text, depositing_user text, date_deposited timestamptz,
		search_text text NOT NULL, imported_at timestamptz NOT NULL DEFAULT now()
	)`)
	if err != nil {
		t.Fatal(err)
	}

	store := New(pool)
	document := records.Document{URI: "u1", SourceYear: 2024, Title: "title", Authors: []string{}, SearchText: "title title"}
	first, err := store.ImportFile(ctx, []records.Document{document})
	if err != nil || first.Inserted != 1 || first.Updated != 0 {
		t.Fatalf("first import: counts=%+v error=%v", first, err)
	}
	second, err := store.ImportFile(ctx, []records.Document{document})
	if err != nil || second.Inserted != 0 || second.Updated != 1 {
		t.Fatalf("second import: counts=%+v error=%v", second, err)
	}

	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM documents").Scan(&count); err != nil || count != 1 {
		t.Fatalf("count=%d error=%v", count, err)
	}

	invalid := document
	invalid.URI = "u3"
	invalid.SourceYear = 2201
	valid := document
	valid.URI = "u2"
	if _, err := store.ImportFile(ctx, []records.Document{valid, invalid}); err == nil {
		t.Fatal("expected transaction error")
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM documents WHERE uri = 'u2'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("rolled-back count=%d error=%v", count, err)
	}
}

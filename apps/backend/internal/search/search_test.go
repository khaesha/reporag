package search

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khaesha/reporag/apps/backend/internal/records"
	"github.com/khaesha/reporag/apps/backend/internal/store"
)

func TestFuseAndSort(t *testing.T) {
	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	lexical := []store.SearchCandidate{
		{Document: store.SearchDocument{URI: "lexical", Title: "Zulu"}, Rank: 1},
		{Document: store.SearchDocument{URI: "shared", Title: "Alpha", DateDeposited: &date}, Rank: 2, Exact: true},
	}
	semantic := []store.SearchCandidate{
		{Document: store.SearchDocument{URI: "shared", Title: "Alpha", DateDeposited: &date}, Rank: 1},
		{Document: store.SearchDocument{URI: "semantic", Title: "Beta"}, Rank: 2},
	}
	fused := fuse(lexical, semantic)
	if len(fused) != 3 || fused[0].Document.URI != "shared" || !fused[0].Exact {
		t.Fatalf("fused=%+v", fused)
	}
	titles := results(fused, store.SearchParams{Sort: "title", Page: 1, Limit: 10})
	if titles.Documents[0].URI != "shared" || titles.Documents[1].URI != "semantic" || titles.Documents[2].URI != "lexical" {
		t.Fatalf("title order=%+v", titles.Documents)
	}
	page := results(fused, store.SearchParams{Sort: "relevance", Page: 2, Limit: 2})
	if page.Total != 3 || len(page.Documents) != 1 {
		t.Fatalf("page=%+v", page)
	}
}

func TestHybridFallsBackToLexical(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `CREATE TEMP TABLE documents (
		uri text PRIMARY KEY, source_year smallint, title text NOT NULL, abstract text,
		authors text[] NOT NULL, item_type text, subjects text, divisions text, depositing_user text,
		date_deposited timestamptz, search_text text NOT NULL,
		search_vector tsvector GENERATED ALWAYS AS (to_tsvector('simple', search_text)) STORED,
		imported_at timestamptz NOT NULL DEFAULT now(), embedding extensions.vector(1536),
		embedding_model text, embedding_input_hash text, embedding_dimensions smallint, embedded_at timestamptz
	)`); err != nil {
		t.Fatal(err)
	}
	database := store.New(pool)
	if _, err := database.ImportFile(ctx, []records.Document{{URI: "u1", SourceYear: 2024, Title: "Robot", Authors: []string{}, SearchText: "Robot robot"}}); err != nil {
		t.Fatal(err)
	}
	service := &Service{database: database, embed: func(context.Context, []string) ([][]float32, error) { return nil, errors.New("down") }, sem: make(chan struct{}, 4)}
	result, err := service.Search(ctx, store.SearchParams{Query: "robot", Mode: "hybrid", Sort: "relevance", Page: 1, Limit: 10})
	if err != nil || !result.Degraded || len(result.Documents) != 1 || result.Documents[0].URI != "u1" {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}

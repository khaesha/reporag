package store

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khaesha/reporag/apps/backend/internal/records"
)

func openTestStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, `CREATE TEMP TABLE documents (
		uri text PRIMARY KEY, source_year smallint CHECK (source_year BETWEEN 1900 AND 2100),
		title text NOT NULL, abstract text, authors text[] NOT NULL, item_type text,
		subjects text, divisions text, depositing_user text, date_deposited timestamptz,
		search_text text NOT NULL,
		search_vector tsvector GENERATED ALWAYS AS (to_tsvector('simple', search_text)) STORED,
		imported_at timestamptz NOT NULL DEFAULT now()
	)`)
	if err != nil {
		t.Fatal(err)
	}
	return New(pool), ctx
}

func TestImportFileIsIdempotentAndTransactional(t *testing.T) {
	store, ctx := openTestStore(t)
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
	if err := store.pool.QueryRow(ctx, "SELECT count(*) FROM documents").Scan(&count); err != nil || count != 1 {
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
	if err := store.pool.QueryRow(ctx, "SELECT count(*) FROM documents WHERE uri = 'u2'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("rolled-back count=%d error=%v", count, err)
	}
}

func TestSearchFiltersSortsAndPagination(t *testing.T) {
	store, ctx := openTestStore(t)
	date2024 := time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)
	date2023 := time.Date(2023, time.January, 2, 0, 0, 0, 0, time.UTC)
	abstract := "available"
	thesis := "Thesis"
	article := "Article"
	computerScience := "Computer Science"
	mathematics := "Mathematics"
	documents := []records.Document{
		{URI: "u-beta", SourceYear: 2024, Title: "Beta common", Abstract: &abstract, Authors: []string{"Adi Kurniawan"}, ItemType: &thesis, Divisions: &computerScience, DateDeposited: &date2024, SearchText: "Beta common Beta common Adi Kurniawan available"},
		{URI: "u-alpha", SourceYear: 2023, Title: "Alpha common", Authors: []string{"Bob"}, ItemType: &article, Divisions: &mathematics, SearchText: "Alpha common Alpha common Bob"},
		{URI: "u-gamma", SourceYear: 2022, Title: "Gamma common", Abstract: &abstract, Authors: []string{"Carol"}, ItemType: &thesis, Divisions: &computerScience, DateDeposited: &date2023, SearchText: "Gamma common Gamma common Carol available"},
		{URI: "u-title", SourceYear: 2021, Title: "Exact Search Title", Authors: []string{"Dina"}, ItemType: &thesis, Divisions: &computerScience, SearchText: "Exact Search Title Exact Search Title Dina"},
		{URI: "u-abstract", SourceYear: 2020, Title: "Unrelated", Abstract: &abstract, Authors: []string{"Eka"}, ItemType: &thesis, Divisions: &computerScience, SearchText: "Unrelated Unrelated Exact Search Title"},
		{URI: "u-missing", SourceYear: 2019, Title: "Missing metadata", Authors: []string{}, SearchText: "Missing metadata Missing metadata"},
	}
	if _, err := store.ImportFile(ctx, documents); err != nil {
		t.Fatal(err)
	}

	result, err := store.Search(ctx, SearchParams{Query: "Exact Search Title", Sort: "relevance", Page: 1, Limit: 10})
	if err != nil || len(result.Documents) != 2 || result.Documents[0].URI != "u-title" {
		t.Fatalf("title ranking: result=%+v error=%v", result, err)
	}
	result, err = store.Search(ctx, SearchParams{Query: "Adi Kurniawan", Sort: "relevance", Page: 1, Limit: 10})
	if err != nil || len(result.Documents) != 1 || result.Documents[0].URI != "u-beta" {
		t.Fatalf("author ranking: result=%+v error=%v", result, err)
	}
	result, err = store.Search(ctx, SearchParams{Query: "nonexistenttoken", Sort: "relevance", Page: 1, Limit: 10})
	if err != nil || result.Total != 0 || len(result.Documents) != 0 {
		t.Fatalf("empty search: result=%+v error=%v", result, err)
	}

	year := 2024
	hasAbstractTrue := true
	hasAbstract := false
	filterCases := []struct {
		name   string
		params SearchParams
		want   string
		total  int
	}{
		{name: "year", params: SearchParams{Query: "common", Year: &year, Sort: "relevance", Page: 1, Limit: 10}, want: "u-beta", total: 1},
		{name: "division", params: SearchParams{Query: "common", Division: mathematics, Sort: "relevance", Page: 1, Limit: 10}, want: "u-alpha", total: 1},
		{name: "item type", params: SearchParams{Query: "common", ItemType: article, Sort: "relevance", Page: 1, Limit: 10}, want: "u-alpha", total: 1},
		{name: "has abstract", params: SearchParams{Query: "common", HasAbstract: &hasAbstractTrue, Sort: "relevance", Page: 1, Limit: 10}, want: "u-beta", total: 2},
	}
	for _, test := range filterCases {
		result, err = store.Search(ctx, test.params)
		if err != nil || result.Total != test.total || len(result.Documents) == 0 || result.Documents[0].URI != test.want {
			t.Fatalf("%s filter: result=%+v error=%v", test.name, result, err)
		}
	}
	result, err = store.Search(ctx, SearchParams{Query: "common", Year: &year, Division: computerScience, ItemType: thesis, Sort: "relevance", Page: 1, Limit: 10})
	if err != nil || result.Total != 1 || result.Documents[0].URI != "u-beta" {
		t.Fatalf("combined filters: result=%+v error=%v", result, err)
	}
	result, err = store.Search(ctx, SearchParams{Query: "common", HasAbstract: &hasAbstract, Sort: "relevance", Page: 1, Limit: 10})
	if err != nil || result.Total != 1 || result.Documents[0].URI != "u-alpha" {
		t.Fatalf("abstract filter: result=%+v error=%v", result, err)
	}
	result, err = store.Search(ctx, SearchParams{Query: "common", Division: "Unknown", Sort: "relevance", Page: 1, Limit: 10})
	if err != nil || result.Total != 0 {
		t.Fatalf("unknown filter: result=%+v error=%v", result, err)
	}
	if _, err := store.Search(ctx, SearchParams{Query: "common", Sort: "title; DROP TABLE documents", Page: 1, Limit: 10}); err == nil {
		t.Fatal("unsupported sort was accepted")
	}

	result, err = store.Search(ctx, SearchParams{Query: "common", Sort: "title", Page: 1, Limit: 2})
	if err != nil || result.Total != 3 || len(result.Documents) != 2 || result.Documents[0].URI != "u-alpha" || result.Documents[1].URI != "u-beta" {
		t.Fatalf("title page one: result=%+v error=%v", result, err)
	}
	result, err = store.Search(ctx, SearchParams{Query: "common", Sort: "title", Page: 2, Limit: 2})
	if err != nil || result.Total != 3 || len(result.Documents) != 1 || result.Documents[0].URI != "u-gamma" {
		t.Fatalf("title page two: result=%+v error=%v", result, err)
	}
	result, err = store.Search(ctx, SearchParams{Query: "common", Sort: "date", Page: 1, Limit: 10})
	if err != nil || len(result.Documents) != 3 || result.Documents[0].URI != "u-beta" || result.Documents[1].URI != "u-gamma" || result.Documents[2].URI != "u-alpha" {
		t.Fatalf("date sort: result=%+v error=%v", result, err)
	}

	filters, err := store.Filters(ctx)
	if err != nil || len(filters.Years) != 6 || filters.Years[0] != 2024 || filters.Years[5] != 2019 || len(filters.Divisions) != 2 || filters.Divisions[0] != computerScience || filters.Divisions[1] != mathematics || len(filters.ItemTypes) != 2 || filters.ItemTypes[0] != article || filters.ItemTypes[1] != thesis {
		t.Fatalf("filters=%+v error=%v", filters, err)
	}
}

func TestCorpusRetrievalEvaluation(t *testing.T) {
	store, ctx := openTestStore(t)
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "../../../.."))
	files, err := records.Discover(filepath.Join(root, "docs/repository-data"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		documents, _, err := records.Read(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.ImportFile(ctx, documents); err != nil {
			t.Fatal(err)
		}
	}

	contents, err := os.ReadFile(filepath.Join(root, "docs/chapter-1/evaluation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var evaluation []struct {
		Query        string   `json:"query"`
		Kind         string   `json:"kind"`
		ExpectedURIs []string `json:"expected_uris"`
		MaxRank      int      `json:"max_rank"`
	}
	if err := json.Unmarshal(contents, &evaluation); err != nil {
		t.Fatal(err)
	}

	passed := 0
	for _, item := range evaluation {
		result, err := store.Search(ctx, SearchParams{Query: item.Query, Sort: "relevance", Page: 1, Limit: item.MaxRank})
		if err != nil {
			t.Fatalf("query %q: %v", item.Query, err)
		}
		matched := false
		for _, document := range result.Documents {
			for _, expectedURI := range item.ExpectedURIs {
				if document.URI == expectedURI {
					matched = true
				}
			}
		}
		if matched {
			passed++
		}
		if item.Kind != "topic" && !matched {
			t.Errorf("%s query %q did not rank an expected URI first", item.Kind, item.Query)
		}
	}
	if passed < 24 {
		t.Fatalf("retrieval evaluation passed %d/%d; want at least 24", passed, len(evaluation))
	}
	t.Logf("retrieval evaluation passed %d/%d", passed, len(evaluation))
}

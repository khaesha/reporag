package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/khaesha/reporag/apps/backend/internal/store"
)

func newTestHandler(
	ping func(context.Context) error,
	search func(context.Context, store.SearchParams) (store.SearchResult, error),
	filters func(context.Context) (store.FilterValues, error),
) http.Handler {
	if search == nil {
		search = func(context.Context, store.SearchParams) (store.SearchResult, error) {
			return store.SearchResult{Documents: []store.SearchDocument{}}, nil
		}
	}
	if filters == nil {
		filters = func(context.Context) (store.FilterValues, error) {
			return store.FilterValues{Years: []int32{}, Divisions: []string{}, ItemTypes: []string{}}, nil
		}
	}
	return New(ping, search, filters, "http://localhost:3000")
}

func TestProbes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		path   string
		ping   func(context.Context) error
		status int
		body   string
	}{
		{name: "health", path: "/healthz", ping: func(context.Context) error { return nil }, status: http.StatusOK, body: `{"status":"ok"}`},
		{name: "ready", path: "/readyz", ping: func(context.Context) error { return nil }, status: http.StatusOK, body: `{"status":"ready"}`},
		{name: "not ready", path: "/readyz", ping: func(context.Context) error { return errors.New("down") }, status: http.StatusServiceUnavailable, body: `{"error":{"code":"not_ready","message":"database unavailable"}}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			newTestHandler(test.ping, nil, nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != test.status || response.Body.String() != test.body {
				t.Fatalf("got %d %q", response.Code, response.Body.String())
			}
			if response.Header().Get("X-Request-ID") == "" {
				t.Fatal("missing request ID")
			}
		})
	}
}

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestHandler(func(context.Context) error { return nil }, nil, nil)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/healthz", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("unexpected preflight response: %d %+v", response.Code, response.Header())
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("Origin", "https://example.com")
	handler.ServeHTTP(response, request)
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("disallowed origin received CORS header")
	}
}

func TestSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var received store.SearchParams
	handler := newTestHandler(
		func(context.Context) error { return nil },
		func(_ context.Context, params store.SearchParams) (store.SearchResult, error) {
			received = params
			return store.SearchResult{
				Total: 1,
				Documents: []store.SearchDocument{{
					Title: "Example", Authors: []string{"Author"}, SourceYear: 2024, URI: "https://example.test/1", Score: 0.5,
				}},
			}, nil
		},
		nil,
	)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=%20machine%20learning%20&year=2024&division=Computer+Science&item_type=Thesis&has_abstract=false&sort=date&page=2&limit=5", nil)
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if received.Query != "machine learning" || received.Year == nil || *received.Year != 2024 || received.Division != "Computer Science" || received.ItemType != "Thesis" || received.HasAbstract == nil || *received.HasAbstract || received.Sort != "date" || received.Page != 2 || received.Limit != 5 {
		t.Fatalf("unexpected params: %+v", received)
	}
	var body searchResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Query != "machine learning" || body.Page != 2 || body.Limit != 5 || body.Total != 1 || len(body.Results) != 1 {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestSearchDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var received store.SearchParams
	handler := newTestHandler(
		func(context.Context) error { return nil },
		func(_ context.Context, params store.SearchParams) (store.SearchResult, error) {
			received = params
			return store.SearchResult{Documents: []store.SearchDocument{}}, nil
		},
		nil,
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/search?q=robot", nil))
	if response.Code != http.StatusOK || received.Sort != "relevance" || received.Page != 1 || received.Limit != 10 {
		t.Fatalf("status=%d params=%+v", response.Code, received)
	}
	if !strings.Contains(response.Body.String(), `"results":[]`) {
		t.Fatalf("empty results must be an array: %s", response.Body.String())
	}
}

func TestSearchValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []string{
		"", "?q=%20", "?q=" + url.QueryEscape(strings.Repeat("x", 201)),
		"?q=x&year=1899", "?q=x&year=invalid",
		"?q=x&division=%20", "?q=x&division=" + url.QueryEscape(strings.Repeat("x", 201)),
		"?q=x&item_type=%20", "?q=x&has_abstract=1", "?q=x&sort=score",
		"?q=x&page=0", "?q=x&page=invalid", "?q=x&limit=0", "?q=x&limit=51",
	}
	handler := newTestHandler(func(context.Context) error { return nil }, nil, nil)
	for _, query := range tests {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/search"+query, nil))
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_query"`) {
			t.Errorf("query=%q status=%d body=%s", query, response.Code, response.Body.String())
		}
	}
}

func TestSearchAndFilterErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestHandler(
		func(context.Context) error { return nil },
		func(context.Context, store.SearchParams) (store.SearchResult, error) {
			return store.SearchResult{}, errors.New("database error")
		},
		func(context.Context) (store.FilterValues, error) {
			return store.FilterValues{}, errors.New("database error")
		},
	)
	for _, path := range []string{"/api/v1/search?q=robot", "/api/v1/filters"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), `"code":"internal_error"`) {
			t.Errorf("path=%s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
}

func TestFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestHandler(
		func(context.Context) error { return nil }, nil,
		func(context.Context) (store.FilterValues, error) {
			return store.FilterValues{Years: []int32{2026, 2025}, Divisions: []string{"Computer Science"}, ItemTypes: []string{"Thesis"}}, nil
		},
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/filters", nil))
	if response.Code != http.StatusOK || response.Body.String() != `{"years":[2026,2025],"divisions":["Computer Science"],"item_types":["Thesis"]}` {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

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

	"github.com/khaesha/reporag/apps/backend/internal/answer"
	"github.com/khaesha/reporag/apps/backend/internal/search"
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
	related := func(context.Context, store.RelatedParams) (store.RelatedResult, error) {
		return store.RelatedResult{Documents: []store.SearchDocument{}}, nil
	}
	trends := func(context.Context, store.TrendParams) (store.TrendResult, error) {
		return store.TrendResult{ByYear: []store.TrendBucket{}, ByDivision: []store.TrendBucket{}, ByItemType: []store.TrendBucket{}, BySubject: []store.TrendBucket{}}, nil
	}
	answerFn := func(context.Context, answer.Request) (answer.Response, error) {
		return answer.Response{Basis: answer.Basis, Citations: []answer.Citation{}}, nil
	}
	return New(ping, search, related, trends, answerFn, filters, "http://localhost:3000")
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
	if response.Header().Get("Server-Timing") != "model;dur=0.00, retrieval;dur=0.00" {
		t.Fatalf("timing=%q", response.Header().Get("Server-Timing"))
	}
	if received.Query != "machine learning" || received.Mode != "hybrid" || received.Year == nil || *received.Year != 2024 || received.Division != "Computer Science" || received.ItemType != "Thesis" || received.HasAbstract == nil || *received.HasAbstract || received.Sort != "date" || received.Page != 2 || received.Limit != 5 {
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
	if response.Code != http.StatusOK || received.Mode != "hybrid" || received.Sort != "relevance" || received.Page != 1 || received.Limit != 10 {
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
		"?q=x&item_type=%20", "?q=x&has_abstract=1", "?q=x&sort=score", "?q=x&mode=invalid",
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

func TestSemanticSearchError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestHandler(
		func(context.Context) error { return nil },
		func(context.Context, store.SearchParams) (store.SearchResult, error) {
			return store.SearchResult{}, search.ErrSemanticUnavailable
		},
		nil,
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/search?q=robot&mode=semantic", nil))
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"code":"semantic_unavailable"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRelatedAndTrends(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var relatedParams store.RelatedParams
	var trendParams store.TrendParams
	handler := New(
		func(context.Context) error { return nil },
		func(context.Context, store.SearchParams) (store.SearchResult, error) {
			return store.SearchResult{}, nil
		},
		func(_ context.Context, params store.RelatedParams) (store.RelatedResult, error) {
			relatedParams = params
			return store.RelatedResult{SourceURI: params.URI, Documents: []store.SearchDocument{}}, nil
		},
		func(_ context.Context, params store.TrendParams) (store.TrendResult, error) {
			trendParams = params
			return store.TrendResult{ByYear: []store.TrendBucket{}, ByDivision: []store.TrendBucket{}, ByItemType: []store.TrendBucket{}, BySubject: []store.TrendBucket{}}, nil
		},
		func(context.Context, answer.Request) (answer.Response, error) { return answer.Response{}, nil },
		func(context.Context) (store.FilterValues, error) { return store.FilterValues{}, nil },
		"http://localhost:3000",
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/related?uri=https%3A%2F%2Fexample.test%2F1&division=Computer+Science", nil))
	if response.Code != http.StatusOK || relatedParams.URI != "https://example.test/1" || relatedParams.Division != "Computer Science" || relatedParams.Limit != 6 {
		t.Fatalf("status=%d params=%+v", response.Code, relatedParams)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/trends?year=2024&division=Computer+Science", nil))
	if response.Code != http.StatusOK || trendParams.Year == nil || *trendParams.Year != 2024 || trendParams.Division != "Computer Science" {
		t.Fatalf("status=%d params=%+v", response.Code, trendParams)
	}
}

func TestRelatedErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		err    error
		status int
		code   string
	}{
		{store.ErrRelatedNotFound, http.StatusNotFound, "not_found"},
		{store.ErrRelatedEmbeddingMissing, http.StatusConflict, "embedding_unavailable"},
	} {
		handler := New(
			func(context.Context) error { return nil },
			func(context.Context, store.SearchParams) (store.SearchResult, error) {
				return store.SearchResult{}, nil
			},
			func(context.Context, store.RelatedParams) (store.RelatedResult, error) {
				return store.RelatedResult{}, test.err
			},
			func(context.Context, store.TrendParams) (store.TrendResult, error) { return store.TrendResult{}, nil },
			func(context.Context, answer.Request) (answer.Response, error) { return answer.Response{}, nil },
			func(context.Context) (store.FilterValues, error) { return store.FilterValues{}, nil },
			"http://localhost:3000",
		)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/related?uri=https%3A%2F%2Fexample.test%2F1", nil))
		if response.Code != test.status || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
			t.Errorf("status=%d body=%s", response.Code, response.Body.String())
		}
	}
}

func TestAnswer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var received answer.Request
	handler := New(
		func(context.Context) error { return nil },
		func(context.Context, store.SearchParams) (store.SearchResult, error) {
			return store.SearchResult{}, nil
		},
		func(context.Context, store.RelatedParams) (store.RelatedResult, error) {
			return store.RelatedResult{}, nil
		},
		func(context.Context, store.TrendParams) (store.TrendResult, error) { return store.TrendResult{}, nil },
		func(_ context.Context, request answer.Request) (answer.Response, error) {
			received = request
			return answer.Response{Answer: "Supported [1]", Basis: answer.Basis, Citations: []answer.Citation{{ID: 1, Title: "Title", URI: "https://example.test/1"}}, InsufficientEvidence: false}, nil
		},
		func(context.Context) (store.FilterValues, error) { return store.FilterValues{}, nil },
		"http://localhost:3000",
	)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/answer", strings.NewReader(`{"query":"  summarize plants ","year":2024,"division":"Computer Science"}`))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || received.Query != "summarize plants" || received.Year == nil || *received.Year != 2024 || received.Division != "Computer Science" || !strings.Contains(response.Body.String(), `"citations":[`) {
		t.Fatalf("status=%d request=%+v body=%s", response.Code, received, response.Body.String())
	}
	for _, body := range []string{`{"query":""}`, `{"query":"x","evidence":"client supplied"}`, `{"query":"x"}{}`} {
		response = httptest.NewRecorder()
		request = httptest.NewRequest(http.MethodPost, "/api/v1/answer", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Errorf("body=%s status=%d", body, response.Code)
		}
	}
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/answer", strings.NewReader(`{"query":"x"}`))
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("content type status=%d", response.Code)
	}
}

func TestAnswerUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := New(
		func(context.Context) error { return nil },
		func(context.Context, store.SearchParams) (store.SearchResult, error) {
			return store.SearchResult{}, nil
		},
		func(context.Context, store.RelatedParams) (store.RelatedResult, error) {
			return store.RelatedResult{}, nil
		},
		func(context.Context, store.TrendParams) (store.TrendResult, error) { return store.TrendResult{}, nil },
		func(context.Context, answer.Request) (answer.Response, error) {
			return answer.Response{}, answer.ErrUnavailable
		},
		func(context.Context) (store.FilterValues, error) { return store.FilterValues{}, nil },
		"http://localhost:3000",
	)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/answer", strings.NewReader(`{"query":"x"}`))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"code":"answer_unavailable"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
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

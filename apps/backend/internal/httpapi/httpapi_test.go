package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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
			New(test.ping, "http://localhost:3000").ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
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
	handler := New(func(context.Context) error { return nil }, "http://localhost:3000")

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

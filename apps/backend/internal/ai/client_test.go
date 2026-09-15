package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func vector(value float32) []float32 {
	result := make([]float32, EmbeddingDimensions)
	for index := range result {
		result[index] = value
	}
	return result
}

func testClient(server *httptest.Server) *Client {
	client := New("test-key")
	client.endpoint = server.URL
	client.http = server.Client()
	client.wait = func(context.Context, time.Duration) error { return nil }
	return client
}

func TestEmbedValidatesAndRestoresProviderOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatal("missing authorization")
		}
		json.NewEncoder(writer).Encode(map[string]any{"data": []map[string]any{
			{"index": 1, "embedding": vector(2)}, {"index": 0, "embedding": vector(1)},
		}})
	}))
	defer server.Close()
	vectors, err := testClient(server).Embed(context.Background(), []string{"one", "two"})
	if err != nil || vectors[0][0] != 1 || vectors[1][0] != 2 {
		t.Fatalf("vectors=%v error=%v", vectors, err)
	}
}

func TestEmbedRetriesTransientFailure(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if requests == 1 {
			writer.Header().Set("Retry-After", "1")
			writer.WriteHeader(http.StatusTooManyRequests)
			return
		}
		json.NewEncoder(writer).Encode(map[string]any{"data": []map[string]any{{"index": 0, "embedding": vector(1)}}})
	}))
	defer server.Close()
	if _, err := testClient(server).Embed(context.Background(), []string{"one"}); err != nil || requests != 2 {
		t.Fatalf("requests=%d error=%v", requests, err)
	}
}

func TestEmbedRejectsMalformedVector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		json.NewEncoder(writer).Encode(map[string]any{"data": []map[string]any{{"index": 0, "embedding": []float32{1}}}})
	}))
	defer server.Close()
	if _, err := testClient(server).Embed(context.Background(), []string{"one"}); err == nil {
		t.Fatal("expected malformed vector error")
	}
}

package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	EmbeddingModel      = "google/gemini-embedding-2"
	EmbeddingDimensions = 1536
	EmbeddingBatchSize  = 32
	requestTimeout      = 30 * time.Second
)

type Client struct {
	apiKey   string
	endpoint string
	http     *http.Client
	wait     func(context.Context, time.Duration) error
}

func New(apiKey string) *Client {
	return &Client{
		apiKey:   apiKey,
		endpoint: "https://openrouter.ai/api/v1/embeddings",
		http:     &http.Client{},
		wait: func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

type embeddingRequest struct {
	Model      string   `json:"model"`
	Dimensions int      `json:"dimensions"`
	Input      []string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (client *Client) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 || len(inputs) > EmbeddingBatchSize {
		return nil, errors.New("embedding batch size is invalid")
	}
	body, err := json.Marshal(embeddingRequest{Model: EmbeddingModel, Dimensions: EmbeddingDimensions, Input: inputs})
	if err != nil {
		return nil, errors.New("embedding request could not be encoded")
	}

	for attempt := 0; attempt < 3; attempt++ {
		vectors, retry, delay, err := client.request(ctx, body, len(inputs))
		if err == nil {
			return vectors, nil
		}
		if !retry || attempt == 2 {
			return nil, err
		}
		if delay == 0 {
			delay = time.Duration(1<<attempt) * 250 * time.Millisecond
		}
		if err := client.wait(ctx, delay); err != nil {
			return nil, errors.New("embedding request canceled")
		}
	}
	return nil, errors.New("embedding request failed")
}

func (client *Client) request(ctx context.Context, body []byte, count int) ([][]float32, bool, time.Duration, error) {
	requestContext, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, client.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, false, 0, errors.New("embedding request could not be created")
	}
	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := client.http.Do(request)
	if err != nil {
		return nil, true, 0, errors.New("embedding provider unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		retry := response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError
		return nil, retry, retryAfter(response.Header.Get("Retry-After")), errors.New("embedding provider rejected request")
	}

	var decoded embeddingResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, false, 0, errors.New("embedding provider returned invalid JSON")
	}
	vectors, err := validateResponse(decoded, count)
	if err != nil {
		return nil, false, 0, err
	}
	return vectors, false, 0, nil
}

func validateResponse(response embeddingResponse, count int) ([][]float32, error) {
	if len(response.Data) != count {
		return nil, errors.New("embedding provider returned wrong vector count")
	}
	vectors := make([][]float32, count)
	for _, item := range response.Data {
		if item.Index < 0 || item.Index >= count || vectors[item.Index] != nil {
			return nil, errors.New("embedding provider returned invalid vector order")
		}
		if len(item.Embedding) != EmbeddingDimensions {
			return nil, fmt.Errorf("embedding provider returned wrong dimensions")
		}
		for _, value := range item.Embedding {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				return nil, errors.New("embedding provider returned invalid vector values")
			}
		}
		vectors[item.Index] = item.Embedding
	}
	return vectors, nil
}

func retryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	when, err := http.ParseTime(value)
	if err == nil && when.After(time.Now()) {
		return time.Until(when)
	}
	return 0
}

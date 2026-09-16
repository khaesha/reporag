package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	GenerationModel     = "openai/gpt-5.6-luna"
	GenerationInputMax  = 16_000
	GenerationOutputMax = 800
	requestTimeout      = 30 * time.Second
)

type Client struct {
	apiKey             string
	endpoint           string
	generationEndpoint string
	http               *http.Client
	wait               func(context.Context, time.Duration) error
	sem                chan struct{}
}

func New(apiKey string) *Client {
	return &Client{
		apiKey:             apiKey,
		endpoint:           "https://openrouter.ai/api/v1/embeddings",
		generationEndpoint: "https://openrouter.ai/api/v1/chat/completions",
		http:               &http.Client{},
		sem:                make(chan struct{}, 4),
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
	if err := client.acquire(ctx); err != nil {
		return nil, err
	}
	defer client.release()
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

type GenerationResult struct {
	Text             string
	PromptTokens     int
	CompletionTokens int
}

type generationRequest struct {
	Model          string    `json:"model"`
	Messages       []message `json:"messages"`
	MaxTokens      int       `json:"max_tokens"`
	ResponseFormat struct {
		Type string `json:"type"`
	} `json:"response_format"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type generationResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (client *Client) Generate(ctx context.Context, prompt string) (GenerationResult, error) {
	if prompt == "" || len(prompt) > GenerationInputMax {
		return GenerationResult{}, errors.New("generation input is invalid")
	}
	if err := client.acquire(ctx); err != nil {
		return GenerationResult{}, err
	}
	defer client.release()

	payload := generationRequest{
		Model:     GenerationModel,
		Messages:  []message{{Role: "user", Content: prompt}},
		MaxTokens: GenerationOutputMax,
	}
	payload.ResponseFormat.Type = "json_object"
	body, err := json.Marshal(payload)
	if err != nil {
		return GenerationResult{}, errors.New("generation request could not be encoded")
	}
	requestContext, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, client.generationEndpoint, bytes.NewReader(body))
	if err != nil {
		return GenerationResult{}, errors.New("generation request could not be created")
	}
	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := client.http.Do(request)
	if err != nil {
		return GenerationResult{}, errors.New("generation provider unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return GenerationResult{}, errors.New("generation provider rejected request")
	}
	var decoded generationResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 128*1024)).Decode(&decoded); err != nil {
		return GenerationResult{}, errors.New("generation provider returned invalid JSON")
	}
	if len(decoded.Choices) != 1 || strings.TrimSpace(decoded.Choices[0].Message.Content) == "" {
		return GenerationResult{}, errors.New("generation provider returned invalid content")
	}
	return GenerationResult{Text: decoded.Choices[0].Message.Content, PromptTokens: decoded.Usage.PromptTokens, CompletionTokens: decoded.Usage.CompletionTokens}, nil
}

func (client *Client) acquire(ctx context.Context) error {
	select {
	case client.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return errors.New("model request canceled")
	}
}

func (client *Client) release() { <-client.sem }

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

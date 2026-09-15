package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khaesha/reporag/apps/backend/internal/ai"
	"github.com/khaesha/reporag/apps/backend/internal/store"
)

type summary struct {
	Embedded int
	Skipped  int
	Failed   int
	Total    int
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result, err := run(ctx)
	fmt.Printf("embedded=%d skipped=%d failed=%d total=%d\n", result.Embedded, result.Skipped, result.Failed, result.Total)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) (summary, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	apiKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	if databaseURL == "" {
		return summary{}, errors.New("DATABASE_URL is required")
	}
	if apiKey == "" {
		return summary{}, errors.New("OPENROUTER_API_KEY is required")
	}
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return summary{}, errors.New("DATABASE_URL is invalid")
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return summary{}, errors.New("database pool initialization failed")
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return summary{}, errors.New("database unavailable")
	}

	database := store.New(pool)
	documents, err := database.EmbeddingDocuments(ctx)
	if err != nil {
		return summary{}, err
	}
	result := summary{Total: len(documents)}
	type candidate struct {
		uri   string
		hash  string
		input string
	}
	pending := make([]candidate, 0)
	for _, document := range documents {
		hash := document.Document.EmbeddingInputHash()
		if document.HasEmbedding && document.EmbeddingModel != nil && *document.EmbeddingModel == ai.EmbeddingModel &&
			document.EmbeddingInputHash != nil && *document.EmbeddingInputHash == hash &&
			document.EmbeddingDimensions != nil && int(*document.EmbeddingDimensions) == ai.EmbeddingDimensions {
			result.Skipped++
			continue
		}
		pending = append(pending, candidate{uri: document.Document.URI, hash: hash, input: document.Document.EmbeddingInput()})
	}

	client := ai.New(apiKey)
	for start := 0; start < len(pending); start += ai.EmbeddingBatchSize {
		end := min(start+ai.EmbeddingBatchSize, len(pending))
		batch := pending[start:end]
		inputs := make([]string, len(batch))
		for index, item := range batch {
			inputs[index] = item.input
		}
		vectors, err := client.Embed(ctx, inputs)
		if err != nil {
			for _, item := range batch {
				fmt.Fprintf(os.Stderr, "failed uri=%s error=%s\n", item.uri, err)
				result.Failed++
			}
			continue
		}
		updates := make([]store.EmbeddingUpdate, len(batch))
		for index, item := range batch {
			updates[index] = store.EmbeddingUpdate{
				URI: item.uri, Vector: vectors[index], Model: ai.EmbeddingModel,
				InputHash: item.hash, Dimensions: ai.EmbeddingDimensions,
			}
		}
		if err := database.UpdateEmbeddings(ctx, updates); err != nil {
			for _, item := range batch {
				fmt.Fprintf(os.Stderr, "failed uri=%s error=embedding storage failed\n", item.uri)
				result.Failed++
			}
			continue
		}
		result.Embedded += len(updates)
	}
	if result.Failed != 0 {
		return result, errors.New("embedding failures remain")
	}
	return result, nil
}

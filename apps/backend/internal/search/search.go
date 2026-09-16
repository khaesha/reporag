package search

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/khaesha/reporag/apps/backend/internal/ai"
	"github.com/khaesha/reporag/apps/backend/internal/store"
)

const candidateLimit = 50

var ErrSemanticUnavailable = errors.New("semantic search unavailable")

type Service struct {
	database *store.Store
	embed    func(context.Context, []string) ([][]float32, error)
}

func New(database *store.Store, client *ai.Client) *Service {
	return &Service{database: database, embed: client.Embed}
}

func (service *Service) Search(ctx context.Context, params store.SearchParams) (store.SearchResult, error) {
	started := time.Now()
	if params.Mode == "" || params.Mode == "lexical" {
		result, err := service.database.Search(ctx, params)
		result.SearchTime = time.Since(started)
		return result, err
	}
	if params.Mode == "semantic" {
		vector, modelTime, err := service.embedding(ctx, params.Query)
		if err != nil {
			return store.SearchResult{}, ErrSemanticUnavailable
		}
		candidates, err := service.database.SemanticCandidates(ctx, params, vector, ai.EmbeddingModel, ai.EmbeddingDimensions, candidateLimit)
		if err != nil {
			return store.SearchResult{}, err
		}
		result := results(candidates, params)
		result.ModelTime = modelTime
		result.SearchTime = time.Since(started) - modelTime
		return result, nil
	}

	lexicalResults := make(chan struct {
		candidates []store.SearchCandidate
		err        error
	}, 1)
	go func() {
		candidates, err := service.database.LexicalCandidates(ctx, params, candidateLimit)
		lexicalResults <- struct {
			candidates []store.SearchCandidate
			err        error
		}{candidates, err}
	}()
	vector, modelTime, embeddingErr := service.embedding(ctx, params.Query)
	lexical := <-lexicalResults
	if lexical.err != nil {
		return store.SearchResult{}, lexical.err
	}
	if embeddingErr != nil {
		result := results(lexical.candidates, params)
		result.Degraded = true
		result.SearchTime = time.Since(started)
		return result, nil
	}
	semantic, err := service.database.SemanticCandidates(ctx, params, vector, ai.EmbeddingModel, ai.EmbeddingDimensions, candidateLimit)
	if err != nil {
		return store.SearchResult{}, err
	}
	result := results(fuse(lexical.candidates, semantic), params)
	result.ModelTime = modelTime
	result.SearchTime = time.Since(started) - modelTime
	return result, nil
}

func (service *Service) embedding(ctx context.Context, query string) ([]float32, time.Duration, error) {
	started := time.Now()
	vectors, err := service.embed(ctx, []string{query})
	if err != nil || len(vectors) != 1 {
		return nil, time.Since(started), ErrSemanticUnavailable
	}
	return vectors[0], time.Since(started), nil
}

func fuse(lexical, semantic []store.SearchCandidate) []store.SearchCandidate {
	byURI := make(map[string]store.SearchCandidate, len(lexical)+len(semantic))
	for _, candidates := range [][]store.SearchCandidate{lexical, semantic} {
		for _, candidate := range candidates {
			current, ok := byURI[candidate.Document.URI]
			if !ok {
				current = candidate
				current.Document.Score = 0
			}
			current.Document.Score += 1 / float32(60+candidate.Rank)
			current.Exact = current.Exact || candidate.Exact
			byURI[candidate.Document.URI] = current
		}
	}
	result := make([]store.SearchCandidate, 0, len(byURI))
	for _, candidate := range byURI {
		result = append(result, candidate)
	}
	return ordered(result, "relevance")
}

func results(candidates []store.SearchCandidate, params store.SearchParams) store.SearchResult {
	orderedCandidates := ordered(candidates, params.Sort)
	start := (params.Page - 1) * params.Limit
	if start >= len(orderedCandidates) {
		return store.SearchResult{Total: len(orderedCandidates), Documents: []store.SearchDocument{}}
	}
	end := min(start+params.Limit, len(orderedCandidates))
	documents := make([]store.SearchDocument, end-start)
	for index, candidate := range orderedCandidates[start:end] {
		documents[index] = candidate.Document
	}
	return store.SearchResult{Total: len(orderedCandidates), Documents: documents}
}

func ordered(candidates []store.SearchCandidate, mode string) []store.SearchCandidate {
	result := append([]store.SearchCandidate(nil), candidates...)
	sort.Slice(result, func(left, right int) bool {
		if mode == "relevance" && result[left].Exact != result[right].Exact {
			return result[left].Exact
		}
		a, b := result[left].Document, result[right].Document
		switch mode {
		case "title":
			if strings.ToLower(a.Title) != strings.ToLower(b.Title) {
				return strings.ToLower(a.Title) < strings.ToLower(b.Title)
			}
			if a.Title != b.Title {
				return a.Title < b.Title
			}
		case "date":
			if a.DateDeposited == nil || b.DateDeposited == nil {
				if a.DateDeposited != nil || b.DateDeposited != nil {
					return a.DateDeposited != nil
				}
			} else if !a.DateDeposited.Equal(*b.DateDeposited) {
				return a.DateDeposited.After(*b.DateDeposited)
			}
		default:
			if a.Score != b.Score {
				return a.Score > b.Score
			}
		}
		if strings.ToLower(a.Title) != strings.ToLower(b.Title) {
			return strings.ToLower(a.Title) < strings.ToLower(b.Title)
		}
		return a.URI < b.URI
	})
	return result
}

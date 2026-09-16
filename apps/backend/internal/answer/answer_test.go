package answer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/khaesha/reporag/apps/backend/internal/ai"
	"github.com/khaesha/reporag/apps/backend/internal/store"
)

func TestAnswerUsesServerEvidence(t *testing.T) {
	abstract := "Evidence about plants."
	var prompt string
	service := &Service{
		search: func(context.Context, store.SearchParams) (store.SearchResult, error) {
			return store.SearchResult{Documents: []store.SearchDocument{{Title: "Plant thesis", Abstract: &abstract, SourceYear: 2024, URI: "u1"}}}, nil
		},
		generate: func(_ context.Context, value string) (ai.GenerationResult, error) {
			prompt = value
			return ai.GenerationResult{Text: `{"answer":"Plants are discussed [1]","citation_ids":[1],"insufficient_evidence":false}`, PromptTokens: 10, CompletionTokens: 5}, nil
		},
	}
	response, err := service.Answer(context.Background(), Request{Query: "plants"})
	if err != nil || response.Answer == "" || len(response.Citations) != 1 || response.Citations[0].URI != "u1" || !strings.Contains(prompt, "untrusted data") || !strings.Contains(prompt, "Evidence about plants.") {
		t.Fatalf("response=%+v error=%v prompt=%q", response, err, prompt)
	}
}

func TestAnswerInsufficientWithoutAbstract(t *testing.T) {
	called := false
	service := &Service{
		search: func(context.Context, store.SearchParams) (store.SearchResult, error) {
			return store.SearchResult{Documents: []store.SearchDocument{{Title: "No abstract", URI: "u1"}}}, nil
		},
		generate: func(context.Context, string) (ai.GenerationResult, error) {
			called = true
			return ai.GenerationResult{}, nil
		},
	}
	response, err := service.Answer(context.Background(), Request{Query: "plants"})
	if err != nil || !response.InsufficientEvidence || called || len(response.Citations) != 0 {
		t.Fatalf("response=%+v error=%v called=%t", response, err, called)
	}
}

func TestAnswerRejectsInvalidModelOutput(t *testing.T) {
	abstract := "Evidence"
	for _, output := range []string{
		`not json`,
		`{"answer":"Claim [2]","citation_ids":[1],"insufficient_evidence":false}`,
		`{"answer":"Claim [1]","citation_ids":[2],"insufficient_evidence":false}`,
		`{"answer":"Claim","citation_ids":[],"insufficient_evidence":false}`,
		`{"answer":"No evidence","citation_ids":[],"insufficient_evidence":true}`,
	} {
		service := &Service{
			search: func(context.Context, store.SearchParams) (store.SearchResult, error) {
				return store.SearchResult{Documents: []store.SearchDocument{{Title: "Title", Abstract: &abstract, URI: "u1"}}}, nil
			},
			generate: func(context.Context, string) (ai.GenerationResult, error) {
				return ai.GenerationResult{Text: output}, nil
			},
		}
		if _, err := service.Answer(context.Background(), Request{Query: "x"}); !errors.Is(err, ErrUnavailable) {
			t.Errorf("output=%s error=%v", output, err)
		}
	}
}

func TestAnswerHandlesModelInsufficiencyAndFailure(t *testing.T) {
	abstract := "Evidence"
	search := func(context.Context, store.SearchParams) (store.SearchResult, error) {
		return store.SearchResult{Documents: []store.SearchDocument{{Title: "Title", Abstract: &abstract, URI: "u1"}}}, nil
	}
	insufficientService := &Service{
		search: search,
		generate: func(context.Context, string) (ai.GenerationResult, error) {
			return ai.GenerationResult{Text: `{"answer":"","citation_ids":[],"insufficient_evidence":true}`}, nil
		},
	}
	response, err := insufficientService.Answer(context.Background(), Request{Query: "x"})
	if err != nil || !response.InsufficientEvidence {
		t.Fatalf("response=%+v error=%v", response, err)
	}
	for _, failure := range []error{errors.New("provider down"), context.DeadlineExceeded} {
		service := &Service{
			search:   search,
			generate: func(context.Context, string) (ai.GenerationResult, error) { return ai.GenerationResult{}, failure },
		}
		if _, err := service.Answer(context.Background(), Request{Query: "x"}); !errors.Is(err, ErrUnavailable) {
			t.Errorf("failure=%v error=%v", failure, err)
		}
	}
}

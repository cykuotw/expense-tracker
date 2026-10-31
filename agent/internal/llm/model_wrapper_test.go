package llm

import (
	"context"
	"errors"
	"iter"
	"strings"
	"testing"

	"expense-tracker/agent/internal/models"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

type countingModel struct {
	calls       int
	err         error
	name        string
	outputLimit int32
}

func (provider *countingModel) Name() string {
	if provider.name != "" {
		return provider.name
	}
	return "fixture"
}

func (provider *countingModel) GenerateContent(
	_ context.Context,
	request *model.LLMRequest,
	_ bool,
) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		provider.calls++
		provider.outputLimit = request.Config.MaxOutputTokens
		yield(&model.LLMResponse{}, provider.err)
	}
}

func TestLimitsAndNoProviderFallback(t *testing.T) {
	provider := &countingModel{err: errors.New("provider failed")}
	wrapped := WithLimitsAndRecording(provider, testLimits(), 2048)
	ctx := WithRunBudget(t.Context(), testLimits())

	for range 6 {
		for _, err := range wrapped.GenerateContent(ctx, &model.LLMRequest{}, false) {
			if !errors.Is(err, provider.err) {
				t.Fatalf("provider error replaced: %v", err)
			}
		}
	}
	for _, err := range wrapped.GenerateContent(ctx, &model.LLMRequest{}, false) {
		if !errors.Is(err, models.ErrBudget) {
			t.Fatalf("missing budget: %v", err)
		}
	}
	if provider.calls != 6 {
		t.Fatalf("provider called %d times", provider.calls)
	}

	oversized := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText(strings.Repeat("x", 65<<10), "user"),
		},
	}
	for _, err := range wrapped.GenerateContent(WithRunBudget(t.Context(), testLimits()), oversized, false) {
		if !errors.Is(err, models.ErrBudget) {
			t.Fatalf("oversized context accepted: %v", err)
		}
	}
	if provider.calls != 6 {
		t.Fatal("oversized request reached provider")
	}
}

func TestModelOutputLimitsAtProviderBoundary(t *testing.T) {
	for _, test := range []struct {
		name                 string
		cap, requested, want int32
	}{
		{"configured default", 2048, 0, 2048},
		{"larger configured cap", 32768, 0, 32768},
		{"configured cap enforced", 32768, 65536, 32768},
		{"smaller cap", 512, 32768, 512},
		{"preserve lower request", 32768, 1024, 1024},
		{"negative request", 32768, -1, 32768},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider := &countingModel{name: "operator-configured-model"}
			request := &model.LLMRequest{Model: "different-request-model", Config: &genai.GenerateContentConfig{MaxOutputTokens: test.requested}}
			for _, err := range WithLimitsAndRecording(provider, testLimits(), test.cap).GenerateContent(WithRunBudget(t.Context(), testLimits()), request, false) {
				if err != nil {
					t.Fatal(err)
				}
			}
			if provider.calls != 1 || provider.outputLimit != test.want {
				t.Fatalf("provider calls=%d cap=%d want=%d", provider.calls, provider.outputLimit, test.want)
			}
		})
	}
}

func TestConfiguredContextLimit(t *testing.T) {
	limits := testLimits()
	limits.MaxContextBytes = 10
	provider := &countingModel{}
	request := &model.LLMRequest{Contents: []*genai.Content{genai.NewContentFromText("hello", "user")}}
	for _, err := range WithLimitsAndRecording(provider, limits, 2048).GenerateContent(WithRunBudget(t.Context(), limits), request, false) {
		if !errors.Is(err, models.ErrBudget) {
			t.Fatalf("custom context ceiling was ignored: %v", err)
		}
	}
	if provider.calls != 0 {
		t.Fatal("oversized context reached the provider")
	}
}

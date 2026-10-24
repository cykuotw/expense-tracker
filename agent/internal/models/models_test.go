package models

import (
	"context"
	"errors"
	"iter"
	"strings"
	"testing"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

func TestResolution(t *testing.T) {
	preferences := []string{
		"",
		" ",
		"unknown",
		"gemini-3.8-flash;override",
		DefaultModel,
		RouterModel,
	}
	for _, preference := range preferences {
		t.Run(preference, func(t *testing.T) {
			resolution, err := Resolve(preference, DefaultModel)
			if err != nil {
				t.Fatal(err)
			}

			approvedPreference := preference == DefaultModel || preference == RouterModel
			if resolution.Fallback == approvedPreference || resolution.Requested != preference {
				t.Fatalf("unexpected resolution: %+v", resolution)
			}
			if !approvedPreference && (resolution.Resolved != DefaultModel || resolution.Reason == "") {
				t.Fatalf("unexpected fallback: %+v", resolution)
			}
		})
	}
	if _, err := Resolve(DefaultModel, "unknown"); err == nil {
		t.Fatal("invalid default accepted")
	}
}

type countingModel struct {
	calls int
	err   error
}

func (*countingModel) Name() string {
	return DefaultModel
}

func (provider *countingModel) GenerateContent(
	context.Context,
	*model.LLMRequest,
	bool,
) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		provider.calls++
		yield(&model.LLMResponse{}, provider.err)
	}
}

func TestLimitsAndNoProviderFallback(t *testing.T) {
	provider := &countingModel{err: errors.New("provider failed")}
	limitedModel := Limited(provider)
	ctx := WithBudget(t.Context())

	for range 6 {
		for _, err := range limitedModel.GenerateContent(ctx, &model.LLMRequest{}, false) {
			if !errors.Is(err, provider.err) {
				t.Fatalf("provider error replaced: %v", err)
			}
		}
	}
	for _, err := range limitedModel.GenerateContent(ctx, &model.LLMRequest{}, false) {
		if !errors.Is(err, ErrBudget) {
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
	for _, err := range limitedModel.GenerateContent(WithBudget(t.Context()), oversized, false) {
		if !errors.Is(err, ErrBudget) {
			t.Fatalf("oversized context accepted: %v", err)
		}
	}
	if provider.calls != 6 {
		t.Fatal("oversized request reached provider")
	}
}

func TestGeminiConstructionWithoutInference(t *testing.T) {
	factory := GeminiFactory(genai.ClientConfig{
		APIKey:  "synthetic-not-a-real-key",
		Backend: genai.BackendGeminiAPI,
	})
	provider, err := factory(t.Context(), "get-balance", DefaultModel)
	if err != nil {
		t.Fatal(err)
	}
	if provider.Name() != DefaultModel {
		t.Fatal(provider.Name())
	}
	if _, err = factory(t.Context(), "get-balance", "unknown"); err == nil {
		t.Fatal("unapproved model accepted")
	}
}

type responseModel struct {
	response *model.LLMResponse
}

func (responseModel) Name() string {
	return "fixture"
}

func (provider responseModel) GenerateContent(
	context.Context,
	*model.LLMRequest,
	bool,
) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		yield(provider.response, nil)
	}
}

func TestResponseLimitsAndThoughtExclusion(t *testing.T) {
	raw := &model.LLMResponse{
		Content: &genai.Content{
			Role: "model",
			Parts: []*genai.Part{
				{Text: "private reasoning", Thought: true},
				{Text: "visible answer"},
			},
		},
		CustomMetadata: map[string]any{"private": "debug"},
	}
	ctx := WithBudget(t.Context())
	provider := Limited(responseModel{response: raw})
	responses := provider.GenerateContent(ctx, &model.LLMRequest{}, true)
	for response, err := range responses {
		if err != nil {
			t.Fatal(err)
		}
		if len(response.Content.Parts) != 1 {
			t.Fatal("private reasoning retained")
		}
		if response.Content.Parts[0].Text != "visible answer" || response.CustomMetadata != nil {
			t.Fatal("private reasoning retained")
		}
	}
	if len(raw.Content.Parts) != 2 {
		t.Fatal("provider response mutated")
	}

	oversized := &model.LLMResponse{
		Content: genai.NewContentFromText(strings.Repeat("x", 33<<10), "model"),
	}
	provider = Limited(responseModel{response: oversized})
	responses = provider.GenerateContent(WithBudget(t.Context()), &model.LLMRequest{}, true)
	for _, err := range responses {
		if !errors.Is(err, ErrBudget) {
			t.Fatal("oversized response accepted", err)
		}
	}
}

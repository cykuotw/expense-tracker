package llm

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"iter"
	"strings"
	"testing"

	"expense-tracker/agent/internal/models"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

type responseModel struct {
	response *model.LLMResponse
	count    int
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
		count := provider.count
		if count == 0 {
			count = 1
		}
		for range count {
			if !yield(provider.response, nil) {
				return
			}
		}
	}
}

func TestConfiguredResponseLimits(t *testing.T) {
	response := &model.LLMResponse{Content: genai.NewContentFromText("visible", "model")}
	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"single response", "total bytes", "event count"} {
		t.Run(scenario, func(t *testing.T) {
			limits := testLimits()
			wantDelivered := 1
			switch scenario {
			case "single response":
				limits.MaxResponseBytes = int32(len(payload) - 1)
				wantDelivered = 0
			case "total bytes":
				limits.MaxResponseBytes = int32(len(payload))
				limits.MaxTotalResponseBytes = int32(2*len(payload) - 1)
			case "event count":
				limits.MaxResponseEvents = 1
			}
			wrapped := WithLimitsAndRecording(responseModel{response: response, count: 3}, limits, 2048)
			delivered, rejected := 0, false
			for result, err := range wrapped.GenerateContent(WithRunBudget(t.Context(), limits), &model.LLMRequest{}, true) {
				if err != nil {
					if !errors.Is(err, models.ErrBudget) {
						t.Fatal(err)
					}
					rejected = true
				} else if result != nil {
					delivered++
				}
			}
			if !rejected || delivered != wantDelivered {
				t.Fatalf("custom response limits ignored: delivered=%d rejected=%v", delivered, rejected)
			}
		})
	}
}

func TestResponseLimitsAndThoughtExclusion(t *testing.T) {
	raw := &model.LLMResponse{
		Content: &genai.Content{
			Role: "model",
			Parts: []*genai.Part{
				{Text: "private reasoning", Thought: true},
				{Text: "visible answer"},
				{Thought: true, ThoughtSignature: []byte{1, 2, 3}},
			},
		},
		CustomMetadata: map[string]any{"private": "debug"},
	}
	ctx := WithRunBudget(t.Context(), testLimits())
	provider := WithLimitsAndRecording(responseModel{response: raw}, testLimits(), 2048)
	responses := provider.GenerateContent(ctx, &model.LLMRequest{}, true)
	for response, err := range responses {
		if err != nil {
			t.Fatal(err)
		}
		if len(response.Content.Parts) != 2 {
			t.Fatal("response did not filter unsigned thoughts and preserve signed continuation")
		}
		if response.Content.Parts[0].Text != "visible answer" || response.CustomMetadata != nil {
			t.Fatal("private reasoning retained")
		}
		if !bytes.Equal(response.Content.Parts[1].ThoughtSignature, []byte{1, 2, 3}) {
			t.Fatal("provider thought signature was not preserved")
		}
	}
	if len(raw.Content.Parts) != 3 {
		t.Fatal("provider response mutated")
	}

	oversized := &model.LLMResponse{
		Content: genai.NewContentFromText(strings.Repeat("x", 33<<10), "model"),
	}
	provider = WithLimitsAndRecording(responseModel{response: oversized}, testLimits(), 2048)
	responses = provider.GenerateContent(WithRunBudget(t.Context(), testLimits()), &model.LLMRequest{}, true)
	for _, err := range responses {
		if !errors.Is(err, models.ErrBudget) {
			t.Fatal("oversized response accepted", err)
		}
	}
}

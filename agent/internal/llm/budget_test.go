package llm

import (
	"errors"
	"testing"

	"expense-tracker/agent/internal/models"

	"google.golang.org/adk/v2/model"
)

func TestRunBudgetSharedAcrossModelsAndTools(t *testing.T) {
	ctx := WithRunBudget(t.Context(), testLimits())
	providers := []*countingModel{{}, {}}
	wrapped := []model.LLM{WithLimitsAndRecording(providers[0], testLimits(), 2048), WithLimitsAndRecording(providers[1], testLimits(), 2048)}
	for i := range 6 {
		for _, err := range wrapped[i%2].GenerateContent(ctx, &model.LLMRequest{}, false) {
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, err := range wrapped[0].GenerateContent(ctx, &model.LLMRequest{}, false) {
		if !errors.Is(err, models.ErrBudget) {
			t.Fatalf("model wrappers did not share the turn budget: %v", err)
		}
	}
	if providers[0].calls != 3 || providers[1].calls != 3 {
		t.Fatal("exhausted budget reached a provider")
	}
	for range 2 {
		if err := ConsumeToolCall(ctx); err != nil {
			t.Fatal("model calls consumed the independent tool budget", err)
		}
	}
	if !errors.Is(ConsumeToolCall(ctx), models.ErrBudget) {
		t.Fatal("tool-call limit was not enforced")
	}
	for range 4 {
		if err := ConsumeStateUpdate(ctx); err != nil {
			t.Fatal("tool calls consumed the independent state budget", err)
		}
	}
	if !errors.Is(ConsumeStateUpdate(ctx), models.ErrBudget) {
		t.Fatal("state-update limit was not enforced")
	}
	for _, err := range wrapped[0].GenerateContent(WithRunBudget(t.Context(), testLimits()), &model.LLMRequest{}, false) {
		if err != nil {
			t.Fatal("a new turn inherited the previous budget", err)
		}
	}
}

func TestConfiguredRunBudgets(t *testing.T) {
	limits := testLimits()
	limits.MaxModelCalls, limits.MaxToolCalls, limits.MaxStateUpdates = 1, 3, 2
	ctx := WithRunBudget(t.Context(), limits)
	// Changing the caller's value must not change an already admitted turn.
	limits.MaxModelCalls = 100
	if err := consumeModelCall(ctx); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(consumeModelCall(ctx), models.ErrBudget) {
		t.Fatal("custom model-call ceiling was ignored")
	}
	for range 3 {
		if err := ConsumeToolCall(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if !errors.Is(ConsumeToolCall(ctx), models.ErrBudget) {
		t.Fatal("custom tool-call ceiling was ignored")
	}
	for range 2 {
		if err := ConsumeStateUpdate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if !errors.Is(ConsumeStateUpdate(ctx), models.ErrBudget) {
		t.Fatal("custom state-update ceiling was ignored")
	}
}

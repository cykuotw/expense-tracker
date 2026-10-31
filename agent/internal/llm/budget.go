package llm

import (
	"context"
	"sync/atomic"

	"expense-tracker/agent/internal/models"
)

type runBudget struct {
	limits       models.LLMLimits
	modelCalls   atomic.Int64
	toolCalls    atomic.Int64
	stateUpdates atomic.Int64
}

type runBudgetKey struct{}

// WithRunBudget starts fresh model, tool and state-update counters for one turn.
func WithRunBudget(ctx context.Context, limits models.LLMLimits) context.Context {
	return context.WithValue(ctx, runBudgetKey{}, &runBudget{limits: limits})
}

// consumeModelCall shares the turn's model-call budget across all agents.
func consumeModelCall(ctx context.Context) error {
	budget, exists := ctx.Value(runBudgetKey{}).(*runBudget)
	if !exists || budget.modelCalls.Add(1) > int64(budget.limits.MaxModelCalls) {
		return models.ErrBudget
	}
	return ctx.Err()
}

// ConsumeToolCall charges one tool call against the shared turn budget.
func ConsumeToolCall(ctx context.Context) error {
	budget, exists := ctx.Value(runBudgetKey{}).(*runBudget)
	if !exists || budget.toolCalls.Add(1) > int64(budget.limits.MaxToolCalls) {
		return models.ErrBudget
	}
	return ctx.Err()
}

// ConsumeStateUpdate charges one state change against the shared turn budget.
func ConsumeStateUpdate(ctx context.Context) error {
	budget, exists := ctx.Value(runBudgetKey{}).(*runBudget)
	if !exists || budget.stateUpdates.Add(1) > int64(budget.limits.MaxStateUpdates) {
		return models.ErrBudget
	}
	return ctx.Err()
}

package updatestate

import (
	"context"
	"errors"
	"expense-tracker/agent/internal/fixtures"
	"math"
	"strings"
	"testing"

	"expense-tracker/agent/internal/credentials"
	"expense-tracker/agent/internal/llm"
	"expense-tracker/agent/internal/models"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/session"
)

func stateForTest(t *testing.T) session.State {
	t.Helper()
	created, err := session.InMemoryService().Create(t.Context(), &session.CreateRequest{AppName: "test", UserID: "test"})
	require.NoError(t, err)
	return created.Session.State()
}

func TestUpdateStateGenericPatchAndRedaction(t *testing.T) {
	state := stateForTest(t)
	ctx := credentials.WithSecrets(llm.WithRunBudget(t.Context(), fixtures.Limits()), "synthetic-credential")
	nested := map[string]any{"itemQuery": "milk", "page": 2}
	result, err := UpdateState(ctx, state, models.StateUpdate{Set: map[string]any{
		"filters": nested, "selectedGroupId": "demo", "previousQuery": "old",
	}})
	require.NoError(t, err)
	require.Equal(t, "staged", result.Status)
	nested["itemQuery"] = "changed externally"
	result, err = UpdateState(ctx, state, models.StateUpdate{
		Set: map[string]any{"filters": map[string]any{"itemQuery": "bread"}, "nullable": nil,
			"values": []any{true, "synthetic-credential"}, "apiKey": "private"},
		Unset: []string{"previousQuery", "absent"},
	})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"itemQuery": "bread"}, result.State["filters"])
	require.Equal(t, "demo", result.State["selectedGroupId"])
	require.Contains(t, result.State, "nullable")
	require.Nil(t, result.State["nullable"])
	require.NotContains(t, result.State, "previousQuery")
	require.Equal(t, "[REDACTED]", result.State["apiKey"])
	require.Equal(t, []any{true, "[REDACTED]"}, result.State["values"])
	// State edits use a separate budget from the two backend reads.
	require.NoError(t, llm.ConsumeToolCall(ctx))
	require.NoError(t, llm.ConsumeToolCall(ctx))
	for range 2 {
		_, err = UpdateState(ctx, state, models.StateUpdate{Set: map[string]any{"arbitrary.key": true}})
		require.NoError(t, err)
	}
	_, err = UpdateState(ctx, state, models.StateUpdate{Unset: []string{"nullable"}})
	require.ErrorIs(t, err, models.ErrBudget)
}

func TestUpdateStateRejectsInvalidWithoutMutation(t *testing.T) {
	deep := any(true)
	for range maxStateDepth + 1 {
		deep = []any{deep}
	}
	cycle := map[string]any{}
	cycle["self"] = cycle
	for name, input := range map[string]models.StateUpdate{
		"empty":       {},
		"empty key":   {Set: map[string]any{"": true}},
		"control key": {Unset: []string{"bad\nkey"}},
		"duplicate":   {Unset: []string{"x", "x"}},
		"conflict":    {Set: map[string]any{"x": true}, Unset: []string{"x"}},
		"large":       {Set: map[string]any{"x": strings.Repeat("x", maxStateBytes)}},
		"depth":       {Set: map[string]any{"x": deep}},
		"cycle":       {Set: cycle},
		"nodes":       {Set: map[string]any{"x": make([]any, maxStateNodes)}},
		"non JSON":    {Set: map[string]any{"x": math.NaN()}},
	} {
		t.Run(name, func(t *testing.T) {
			state := stateForTest(t)
			require.NoError(t, state.Set(ContextKey, map[string]any{"original": true}))
			_, err := UpdateState(llm.WithRunBudget(t.Context(), fixtures.Limits()), state, input)
			require.Equal(t, models.Validation, models.PublicCode(err))
			value, err := state.Get(ContextKey)
			require.NoError(t, err)
			require.Equal(t, map[string]any{"original": true}, value)
		})
	}
	state := stateForTest(t)
	ctx, cancel := context.WithCancel(llm.WithRunBudget(t.Context(), fixtures.Limits()))
	cancel()
	_, err := UpdateState(ctx, state, models.StateUpdate{Set: map[string]any{"x": true}})
	require.ErrorIs(t, err, context.Canceled)
	_, err = state.Get(ContextKey)
	require.ErrorIs(t, err, session.ErrStateKeyNotExist)
	_, err = UpdateState(llm.WithRunBudget(t.Context(), fixtures.Limits()), failingState{state}, models.StateUpdate{Set: map[string]any{"x": true}})
	require.Equal(t, models.Internal, models.PublicCode(err))
}

type failingState struct{ session.State }

func (failingState) Set(string, any) error { return errors.New("synthetic write failure") }

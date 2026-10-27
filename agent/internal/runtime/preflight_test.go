package runtime

import (
	"context"
	"encoding/json/v2"
	"iter"
	"strings"
	"testing"
	"time"

	"expense-tracker/agent/internal/clients/worker"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/tools"
	"expense-tracker/internal/agentcontract"
	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
)

type authenticatedBalance struct {
	calls  int
	actor  string
	expiry time.Time
	err    error
}

func (b *authenticatedBalance) Preflight(context.Context) (agentcontract.Preflight, error) {
	b.calls++
	return agentcontract.Preflight{ActorID: b.actor, ExpiresAt: b.expiry}, b.err
}
func (*authenticatedBalance) GetBalance(ctx context.Context, group string) (tools.BalanceResult, error) {
	return (tools.FixtureBalanceClient{}).GetBalance(ctx, group)
}

func TestPreflightBeforeInferenceOnEveryRun(t *testing.T) {
	balance := &authenticatedBalance{actor: "actor-one", expiry: time.Now().Add(time.Hour)}
	modelCalls := 0
	factory := func(ctx context.Context, skill, name string) (model.LLM, error) {
		base, err := models.FixtureFactory(ctx, skill, name)
		return modelFunc(func(ctx context.Context, request *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
			modelCalls++
			payload, marshalErr := json.Marshal(request)
			require.NoError(t, marshalErr)
			require.NotContains(t, string(payload), "request-only-sensitive-access")
			return base.GenerateContent(ctx, request, stream)
		}), err
	}
	store := session.InMemoryService()
	app, err := New(t.Context(), Config{Models: factory, Balance: balance, Sessions: store})
	require.NoError(t, err)
	run := func(message string) (string, error) {
		var visible strings.Builder
		err := app.Run(worker.WithAccessToken(t.Context(), "request-only-sensitive-access"), Request{
			ConversationID: "same-conversation", MessageID: message, Text: "balance demo",
		}, func(output Output) error { visible.WriteString(output.Delta); return nil })
		return visible.String(), err
	}
	balance.err = &tools.Error{Code: tools.Forbidden}
	_, err = run("first")
	require.Equal(t, tools.Forbidden, tools.PublicCode(err))
	require.Zero(t, modelCalls)
	balance.err = nil
	balance.expiry = time.Now().Add(-time.Minute)
	_, err = run("first")
	require.Equal(t, tools.Unauthenticated, tools.PublicCode(err))
	require.Zero(t, modelCalls)
	balance.expiry = time.Now().Add(time.Hour)
	visible, err := run("first")
	require.NoError(t, err)
	require.Contains(t, visible, "12.34")
	require.NotContains(t, visible, "request-only-sensitive-access")
	firstCalls := modelCalls
	balance.err = &tools.Error{Code: tools.Unauthenticated}
	_, err = run("second")
	require.Equal(t, tools.Unauthenticated, tools.PublicCode(err))
	require.Equal(t, firstCalls, modelCalls)
	balance.err = nil
	balance.actor = "actor-two"
	_, err = run("first")
	require.NoError(t, err, "another actor can use the same conversation and message selectors without sharing history")
	require.Equal(t, 5, balance.calls)
	for _, actor := range []string{"actor-one", "actor-two"} {
		stored, err := store.Get(t.Context(), &session.GetRequest{AppName: sessionAppName, UserID: actor, SessionID: "same-conversation"})
		require.NoError(t, err)
		for event := range stored.Session.Events().All() {
			payload, err := json.Marshal(event)
			require.NoError(t, err)
			require.NotContains(t, string(payload), "request-only-sensitive-access")
		}
	}
}

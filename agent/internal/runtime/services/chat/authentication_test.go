package chat_test

import (
	"context"
	"encoding/json/v2"
	"iter"
	"strings"
	"testing"
	"time"

	"expense-tracker/agent/internal/credentials"
	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/runtime"
	"expense-tracker/agent/internal/runtime/engine"
	"expense-tracker/agent/internal/tools"
	"expense-tracker/agent/internal/tools/functiontool/getbalance"
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
func (b *authenticatedBalance) Authenticate(ctx context.Context) (models.Identity, error) {
	actor, err := b.Preflight(ctx)
	return models.Identity{ID: actor.ActorID, ExpiresAt: actor.ExpiresAt}, err
}
func (*authenticatedBalance) GetBalance(ctx context.Context, group string) (getbalance.Result, error) {
	return (fixtures.BalanceClient{}).GetBalance(ctx, group)
}

func TestPreflightBeforeInferenceOnEveryRun(t *testing.T) {
	balance := &authenticatedBalance{actor: "actor-one", expiry: time.Now().Add(time.Hour)}
	modelCalls := 0
	factory := func(ctx context.Context, skill, name string) (model.LLM, error) {
		base, err := fixtures.ModelFactory(ctx, skill, name)
		return modelFunc(func(ctx context.Context, request *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
			modelCalls++
			payload, marshalErr := json.Marshal(request)
			require.NoError(t, marshalErr)
			require.NotContains(t, string(payload), "request-only-sensitive-access")
			return base.GenerateContent(ctx, request, stream)
		}), err
	}
	store := session.InMemoryService()
	app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(), Models: factory, Tools: testCatalog(t, balance), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(), Authenticate: balance.Authenticate, Sessions: store})
	require.NoError(t, err)
	run := func(message string) (string, error) {
		var visible strings.Builder
		err := app.Run(credentials.WithAccessToken(t.Context(), "request-only-sensitive-access"), models.ChatRequest{
			ConversationID: "same-conversation", MessageID: message, Text: "balance demo",
		}, func(output models.Output) error { visible.WriteString(output.Delta); return nil })
		return visible.String(), err
	}
	balance.err = &models.Error{Code: models.Forbidden}
	_, err = run("first")
	require.Equal(t, models.Forbidden, models.PublicCode(err))
	require.Zero(t, modelCalls)
	balance.err = nil
	balance.expiry = time.Now().Add(-time.Minute)
	_, err = run("first")
	require.Equal(t, models.Unauthenticated, models.PublicCode(err))
	require.Zero(t, modelCalls)
	balance.expiry = time.Now().Add(time.Hour)
	visible, err := run("first")
	require.NoError(t, err)
	require.Contains(t, visible, "12.34")
	require.NotContains(t, visible, "request-only-sensitive-access")
	firstCalls := modelCalls
	balance.err = &models.Error{Code: models.Unauthenticated}
	_, err = run("second")
	require.Equal(t, models.Unauthenticated, models.PublicCode(err))
	require.Equal(t, firstCalls, modelCalls)
	balance.err = nil
	balance.actor = "actor-two"
	_, err = run("first")
	require.NoError(t, err, "another actor can use the same conversation and message selectors without sharing history")
	require.Equal(t, 5, balance.calls)
	for _, actor := range []string{"actor-one", "actor-two"} {
		stored, err := store.Get(t.Context(), &session.GetRequest{AppName: engine.AppName, UserID: actor, SessionID: memorySessionID(t, store, actor, "same-conversation")})
		require.NoError(t, err)
		for event := range stored.Session.Events().All() {
			payload, err := json.Marshal(event)
			require.NoError(t, err)
			require.NotContains(t, string(payload), "request-only-sensitive-access")
		}
	}
}

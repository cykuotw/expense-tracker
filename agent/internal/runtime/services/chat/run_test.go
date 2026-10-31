package chat_test

import (
	"testing"

	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/runtime"
	"expense-tracker/agent/internal/runtime/engine"
	"expense-tracker/agent/internal/tools"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/session"
)

func TestClientIDsStaySeparateFromRuntimeSession(t *testing.T) {
	store := session.InMemoryService()
	app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(), Models: fixtures.ModelFactory, Tools: testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(), Sessions: store})
	require.NoError(t, err)
	request := models.ChatRequest{ConversationID: "browser-thread", RunID: "browser-run", MessageID: "one", Text: "balance demo"}
	var outputs []models.Output
	require.NoError(t, app.Run(t.Context(), request, func(out models.Output) error { outputs = append(outputs, out); return nil }))
	for _, out := range outputs {
		require.Equal(t, request.ConversationID, out.ConversationID)
		require.Equal(t, request.RunID, out.RunID)
	}
	internal := memorySessionID(t, store, "local-fixture", request.ConversationID)
	_, err = store.Get(t.Context(), &session.GetRequest{AppName: engine.AppName, UserID: "local-fixture", SessionID: request.ConversationID})
	require.Error(t, err)
	request.MessageID = "two"
	err = app.Run(t.Context(), request, func(models.Output) error { return nil })
	require.Equal(t, models.Limit, models.PublicCode(err), "reusing a UI run must not execute a new turn")
	request.RunID = "browser-run-two"
	require.NoError(t, app.Run(t.Context(), request, func(models.Output) error { return nil }))
	require.Equal(t, internal, memorySessionID(t, store, "local-fixture", request.ConversationID))
}

package engine

import (
	"context"
	"testing"
	"time"
	"uuid"

	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/llm"
	"expense-tracker/agent/internal/models"
	sessionstore "expense-tracker/agent/internal/store/session"
	"expense-tracker/agent/internal/tools"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/session"
)

type diagnosticRepository struct {
	sessionstore.Repository
	kinds []string
}

func (repo *diagnosticRepository) Append(_ context.Context, _ models.Run, entry models.Entry) (string, time.Time, error) {
	repo.kinds = append(repo.kinds, entry.Kind)
	if entry.Kind == "adk.event" {
		return "", time.Time{}, sessionstore.ErrPayload
	}
	return uuid.New().String(), time.Now(), nil
}

func TestTurnKeepsADKHistorySeparateFromDiagnostics(t *testing.T) {
	sessions := session.InMemoryService()
	app, err := New(t.Context(), Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(), Models: fixtures.ModelFactory, Tools: testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(), Sessions: sessions})
	require.NoError(t, err)
	repo := &diagnosticRepository{}
	run := models.Run{Key: models.Key{App: AppName, Owner: "local-fixture", Session: uuid.New().String()}, ID: uuid.New().String()}
	ctx := sessionstore.WithRun(llm.WithRunBudget(t.Context(), fixtures.Limits()), repo, run)
	var outputs []models.Output
	err = app.RunTurn(ctx, run.Key, models.ChatRequest{ConversationID: "thread", MessageID: "message", Text: "hello"}, models.Output{}, func(output models.Output) error {
		outputs = append(outputs, output)
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, outputs)
	require.Contains(t, repo.kinds, "model.request")
	require.NotContains(t, repo.kinds, "adk.event")
	stored, err := sessions.Get(t.Context(), &session.GetRequest{AppName: AppName, UserID: run.Owner, SessionID: run.Session})
	require.NoError(t, err)
	require.Positive(t, stored.Session.Events().Len(), "ADK retains history through its session service")
}

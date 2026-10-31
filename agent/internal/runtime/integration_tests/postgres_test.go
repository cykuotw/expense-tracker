package integration_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json/v2"
	"io"
	"iter"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"expense-tracker/agent/internal/credentials"
	"expense-tracker/agent/internal/databasebootstrap"
	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/runtime"
	sessionstore "expense-tracker/agent/internal/store/session"
	"expense-tracker/agent/internal/tools"
	"expense-tracker/agent/internal/tools/functiontool/getbalance"
	"expense-tracker/internal/agentcontract"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

type fakeBalanceClient struct {
	actor    string
	revoked  atomic.Bool
	changed  atomic.Bool
	getCalls atomic.Int32
}

func (b *fakeBalanceClient) Preflight(context.Context) (agentcontract.Preflight, error) {
	return agentcontract.Preflight{ActorID: b.actor, ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (b *fakeBalanceClient) Authenticate(ctx context.Context) (models.Identity, error) {
	actor, err := b.Preflight(ctx)
	return models.Identity{ID: actor.ActorID, ExpiresAt: actor.ExpiresAt}, err
}
func (b *fakeBalanceClient) GetBalance(ctx context.Context, group string) (getbalance.Result, error) {
	b.getCalls.Add(1)
	if b.revoked.Load() {
		return getbalance.Result{}, &models.Error{Code: models.NotFound}
	}
	result, err := (fixtures.BalanceClient{}).GetBalance(ctx, group)
	if b.changed.Load() && len(result.Balances) > 0 {
		result.Balances[0].Amount = "99.99"
	}
	return result, err
}
func openPostgresTestStore(t *testing.T) (*sessionstore.Store, *sql.DB) {
	t.Helper()
	if os.Getenv("AGENT_TEST_POSTGRES") != "1" {
		t.Skip("requires isolated migrated PostgreSQL")
	}
	admin, err := sql.Open("pgx", os.Getenv("AGENT_STATE_ADMIN_DATABASE_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { admin.Close() })
	require.NoError(t, databasebootstrap.ProvisionRole(t.Context(), admin, "agent_state_test", "synthetic_state_password"))
	store, err := sessionstore.Open(t.Context(), os.Getenv("AGENT_STATE_DATABASE_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store, admin
}
func newPersistentTestApp(t *testing.T, store *sessionstore.Store, balance *fakeBalanceClient) *runtime.App {
	t.Helper()
	app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(), Models: fixtures.ModelFactory, Tools: testCatalog(t, balance), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(), Authenticate: balance.Authenticate, Sessions: store, Records: store})
	require.NoError(t, err)
	return app
}
func TestDurableRestartClarificationMappingAndAccess(t *testing.T) {
	store, admin := openPostgresTestStore(t)
	balance := &fakeBalanceClient{actor: uuid.New().String()}
	conversation := uuid.New().String()
	ctx := credentials.WithAccessToken(t.Context(), "synthetic-request-only-access")
	app := newPersistentTestApp(t, store, balance)
	run := func(app *runtime.App, message, text string) ([]models.Output, error) {
		outputs := []models.Output{}
		err := app.Run(ctx, models.ChatRequest{ConversationID: conversation, RunID: "browser-" + message, MessageID: message, Text: text}, func(o models.Output) error { outputs = append(outputs, o); return nil })
		return outputs, err
	}
	outputs, err := run(app, "one", "balance")
	require.NoError(t, err)
	require.NotEmpty(t, outputs)
	firstRun := outputs[0].RunID
	require.Equal(t, "browser-one", firstRun)
	oldKey, err := store.ResolveThread(ctx, sessionAppName, balance.actor, conversation, false)
	require.NoError(t, err)
	_, err = admin.ExecContext(t.Context(), `UPDATE expense_agent.sessions SET created_at=NOW()-INTERVAL '2 hours',updated_at=NOW()-INTERVAL '2 hours' WHERE session_id=$1`, oldKey.Session)
	require.NoError(t, err)
	_, err = admin.ExecContext(t.Context(), `UPDATE expense_agent.events SET created_at=created_at-INTERVAL '2 hours' WHERE session_id=$1`, oldKey.Session)
	require.NoError(t, err)
	restartedStore, err := sessionstore.Open(t.Context(), os.Getenv("AGENT_STATE_DATABASE_URL"))
	require.NoError(t, err)
	defer restartedStore.Close()
	outputs, err = run(newPersistentTestApp(t, restartedStore, balance), "two", "demo")
	require.NoError(t, err)
	var body strings.Builder
	for _, output := range outputs {
		body.WriteString(output.Delta)
		require.NotEmpty(t, output.EventID)
		require.False(t, output.CreatedAt.IsZero())
		require.Equal(t, outputs[0].MessageID, output.MessageID)
	}
	require.Contains(t, body.String(), "12.34")
	require.NotEqual(t, firstRun, outputs[0].RunID)
	require.Equal(t, "browser-two", outputs[0].RunID)
	restored := newPersistentTestApp(t, store, balance)
	replay, err := restored.ReadRunOutput(ctx, conversation, outputs[0].RunID, "")
	require.NoError(t, err)
	require.Equal(t, outputs, replay)
	_, err = run(restored, "two", "balance demo")
	require.NoError(t, err, "a fresh runtime does not retain the previous process's resend guard")
	_, err = run(restored, "two", "balance demo")
	require.Equal(t, models.Limit, models.PublicCode(err), "the same runtime rejects another resend")
	history, err := restored.ListRuns(ctx, conversation)
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.Equal(t, "completed", history[0].Status)
	other := newPersistentTestApp(t, store, &fakeBalanceClient{actor: uuid.New().String()})
	_, err = other.ListRuns(ctx, conversation)
	require.Equal(t, models.NotFound, models.PublicCode(err))
	var eventCount, invocations int
	key, err := store.ResolveThread(ctx, sessionAppName, balance.actor, conversation, false)
	require.NoError(t, err)
	internalRunID, err := store.ResolveRun(ctx, key, conversation, outputs[0].RunID)
	require.NoError(t, err)
	require.NotEqual(t, conversation, key.Session)
	require.NotEqual(t, outputs[0].RunID, internalRunID)
	require.NoError(t, admin.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM expense_agent.events WHERE run_id=$1`, internalRunID).Scan(&eventCount))
	require.Greater(t, eventCount, len(outputs))
	require.NoError(t, admin.QueryRowContext(t.Context(), `SELECT COUNT(DISTINCT invocation_id) FROM expense_agent.events WHERE run_id=$1 AND invocation_id<>''`, internalRunID).Scan(&invocations))
	require.Positive(t, invocations)
	rows, err := admin.QueryContext(t.Context(), `SELECT event_type,data FROM expense_agent.events WHERE run_id=$1`, internalRunID)
	require.NoError(t, err)
	kinds := map[string]bool{}
	for rows.Next() {
		var kind string
		var payload []byte
		require.NoError(t, rows.Scan(&kind, &payload))
		require.NotContains(t, string(payload), "synthetic-request-only-access")
		kinds[kind] = true
	}
	require.NoError(t, rows.Err())
	rows.Close()
	for _, kind := range []string{"model.request", "model.response", "model.finished", "tool.started", "tool.finished", "adk.event", "output"} {
		require.True(t, kinds[kind], kind)
	}
	balance.changed.Store(true)
	calls := balance.getCalls.Load()
	replay, err = restored.ReadRunOutput(ctx, conversation, outputs[0].RunID, "")
	require.NoError(t, err)
	require.Equal(t, outputs, replay, "saved output remains the historical snapshot")
	require.Equal(t, calls, balance.getCalls.Load(), "history reads never query domain data")
	updated, err := run(restored, "updated", "balance demo")
	require.NoError(t, err)
	var updatedText strings.Builder
	for _, output := range updated {
		updatedText.WriteString(output.Delta)
	}
	require.Contains(t, updatedText.String(), "99.99", "new tool reads can persist changed values in the same session")
	balance.revoked.Store(true)
	calls = balance.getCalls.Load()
	_, err = restored.ListRuns(ctx, conversation)
	require.NoError(t, err, "owners retain previously recorded history after group access changes")
	require.Equal(t, calls, balance.getCalls.Load())
	_, err = run(restored, "revoked", "balance demo")
	require.Equal(t, models.NotFound, models.PublicCode(err), "new tool calls still enforce backend authorization")
	balance.revoked.Store(false)
	require.NoError(t, restored.DeleteConversation(ctx, conversation))
	_, err = run(restored, "three", "balance demo")
	require.Equal(t, models.NotFound, models.PublicCode(err))
	_, err = restored.ReadRunOutput(ctx, conversation, outputs[0].RunID, "")
	require.Equal(t, models.NotFound, models.PublicCode(err))
	data, _ := json.Marshal(history)
	require.NotContains(t, string(data), "synthetic-request-only-access")
}

func TestDurableProviderDiagnosticsAndSignatureContinuation(t *testing.T) {
	store, admin := openPostgresTestStore(t)
	balance := &fakeBalanceClient{actor: uuid.New().String()}
	conversation := uuid.New().String()
	var sawSignature atomic.Bool
	signature := []byte{0, 7, 255, 17, 5}
	factory := func(ctx context.Context, skill, name string) (model.LLM, error) {
		base, err := fixtures.ModelFactory(ctx, skill, name)
		return modelFunc(func(ctx context.Context, request *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
			for _, content := range request.Contents {
				if content != nil {
					for _, part := range content.Parts {
						if part != nil && bytes.Equal(part.ThoughtSignature, signature) {
							sawSignature.Store(true)
						}
						if part != nil {
							require.NotContains(t, part.Text, "diagnostic-only-summary")
						}
					}
				}
			}
			return func(yield func(*model.LLMResponse, error) bool) {
				if !yield(&model.LLMResponse{Partial: true, Content: &genai.Content{Role: "model", Parts: []*genai.Part{{Thought: true, Text: "diagnostic-only-summary request-private-access"}}}}, nil) {
					return
				}
				for response, err := range base.GenerateContent(ctx, request, stream) {
					if response != nil && response.Content != nil {
						for _, part := range response.Content.Parts {
							if part != nil && part.FunctionCall != nil {
								part.ThoughtSignature = signature
							}
						}
					}
					if !yield(response, err) {
						return
					}
				}
			}
		}), err
	}
	create := func() *runtime.App {
		app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(), Models: factory, Tools: testCatalog(t, balance), CompletionChecks: tools.CompletionChecks(),
			RouterTools:          tools.RouterTools(),
			InstructionProviders: tools.InstructionProviders(),
			AfterModelCallbacks:  tools.AfterModelCallbacks(), Authenticate: balance.Authenticate, Sessions: store, Records: store})
		require.NoError(t, err)
		return app
	}
	ctx := credentials.WithAccessToken(t.Context(), "request-private-access")
	for _, message := range []string{"one", "two"} {
		app := create()
		err := app.Run(ctx, models.ChatRequest{ConversationID: conversation, MessageID: message, Text: "balance demo"}, func(o models.Output) error {
			require.NotContains(t, o.Delta, "diagnostic-only-summary")
			require.NotContains(t, o.Delta, "request-private-access")
			return nil
		})
		require.NoError(t, err)
	}
	require.True(t, sawSignature.Load())
	var count int
	require.NoError(t, admin.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM expense_agent.events e JOIN expense_agent.sessions s ON s.session_id=e.session_id WHERE s.owner_id=$1 AND e.event_type='model.response' AND e.data::text LIKE '%diagnostic-only-summary%'`, balance.actor).Scan(&count))
	require.Positive(t, count)
	require.NoError(t, admin.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM expense_agent.events e JOIN expense_agent.sessions s ON s.session_id=e.session_id WHERE s.owner_id=$1 AND e.data::text LIKE '%request-private-access%'`, balance.actor).Scan(&count))
	require.Zero(t, count)
}

func TestDurableStorageFailureStopsInferenceAndRetainsRun(t *testing.T) {
	store, admin := openPostgresTestStore(t)
	balance := &fakeBalanceClient{actor: uuid.New().String()}
	conversation := uuid.New().String()
	_, err := admin.ExecContext(t.Context(), `CREATE OR REPLACE FUNCTION expense_agent.test_reject_model() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.event_type='model.request' THEN RAISE EXCEPTION 'synthetic storage failure'; END IF; RETURN NEW; END $$;
 CREATE TRIGGER test_reject_model BEFORE INSERT ON expense_agent.events FOR EACH ROW EXECUTE FUNCTION expense_agent.test_reject_model()`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := admin.ExecContext(context.Background(), `DROP TRIGGER IF EXISTS test_reject_model ON expense_agent.events; DROP FUNCTION IF EXISTS expense_agent.test_reject_model()`)
		require.NoError(t, err)
	})
	var calls atomic.Int32
	factory := func(ctx context.Context, skill, name string) (model.LLM, error) {
		return modelFunc(func(context.Context, *model.LLMRequest, bool) iter.Seq2[*model.LLMResponse, error] {
			calls.Add(1)
			return func(func(*model.LLMResponse, error) bool) {}
		}), nil
	}
	app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(), Models: factory, Tools: testCatalog(t, balance), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(), Authenticate: balance.Authenticate, Sessions: store, Records: store})
	require.NoError(t, err)
	var outputs []models.Output
	err = app.Run(t.Context(), models.ChatRequest{ConversationID: conversation, MessageID: "one", Text: "balance demo"}, func(o models.Output) error { outputs = append(outputs, o); return nil })
	require.Error(t, err)
	require.Zero(t, calls.Load())
	require.Equal(t, "error", outputs[len(outputs)-1].Kind)
	history, err := app.ListRuns(t.Context(), conversation)
	require.NoError(t, err)
	require.Equal(t, "failed", history[0].Status)
	require.NotContains(t, outputs[len(outputs)-1].ErrorCode, "synthetic storage failure")
}

func TestCancelledRunKeepsSessionAndDetectableTerminalState(t *testing.T) {
	store, _ := openPostgresTestStore(t)
	balance := &fakeBalanceClient{actor: uuid.New().String()}
	conversation := uuid.New().String()
	app := newPersistentTestApp(t, store, balance)
	ctx, cancel := context.WithCancel(t.Context())
	err := app.Run(ctx, models.ChatRequest{ConversationID: conversation, MessageID: "one", Text: "balance demo"}, func(o models.Output) error {
		if o.Kind == "text" {
			cancel()
			return context.Canceled
		}
		return nil
	})
	require.Error(t, err)
	history, err := app.ListRuns(t.Context(), conversation)
	require.NoError(t, err)
	require.Equal(t, "interrupted", history[0].Status)
	replay, err := app.ReadRunOutput(t.Context(), conversation, history[0].ID, "")
	require.NoError(t, err)
	require.NotEmpty(t, replay)
}

func TestCompletedLifecycleSurvivesDisconnectedTerminalDelivery(t *testing.T) {
	store, _ := openPostgresTestStore(t)
	app := newPersistentTestApp(t, store, &fakeBalanceClient{actor: uuid.New().String()})
	thread := uuid.New().String()
	err := app.Run(t.Context(), models.ChatRequest{ConversationID: thread, RunID: "ui-run", MessageID: "one", Text: "balance demo"}, func(out models.Output) error {
		if out.Kind == "finished" {
			return io.ErrClosedPipe
		}
		return nil
	})
	require.ErrorIs(t, err, io.ErrClosedPipe)
	history, err := app.ListRuns(t.Context(), thread)
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, "completed", history[0].Status, "delivery failure must not overwrite a committed result")
	replay, err := app.ReadRunOutput(t.Context(), thread, "ui-run", "")
	require.NoError(t, err)
	require.Equal(t, "finished", replay[len(replay)-1].Kind)
}

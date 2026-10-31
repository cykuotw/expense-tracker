package integration_test

import (
	"context"
	"expense-tracker/agent/internal/fixtures"
	"os"
	"testing"
	"uuid"

	"expense-tracker/agent/internal/runtime"
	sessionstore "expense-tracker/agent/internal/store/session"
	"expense-tracker/agent/internal/tools"
	"expense-tracker/agent/internal/tools/functiontool/updatestate"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/session"
)

func TestUpdateStatePostgresRestartAndRollback(t *testing.T) {
	store, admin := openPostgresTestStore(t)
	balance := &fakeBalanceClient{actor: uuid.New().String()}
	conversation := uuid.New().String()
	appFor := func(store *sessionstore.Store) *runtime.App {
		app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(), Models: stateModelFactory(t), Tools: testCatalog(t, balance), CompletionChecks: tools.CompletionChecks(),
			RouterTools:          tools.RouterTools(),
			InstructionProviders: tools.InstructionProviders(),
			AfterModelCallbacks:  tools.AfterModelCallbacks(), Authenticate: balance.Authenticate, Sessions: store, Records: store})
		require.NoError(t, err)
		return app
	}
	_, _, err := runMessage(t, appFor(store), conversation, "one", "save")
	require.NoError(t, err)
	reopened, err := sessionstore.Open(t.Context(), os.Getenv("AGENT_STATE_DATABASE_URL"))
	require.NoError(t, err)
	defer reopened.Close()
	app := appFor(reopened)
	_, answer, err := runMessage(t, app, conversation, "two", "inspect")
	require.NoError(t, err)
	require.Contains(t, answer, `"itemQuery":"milk"`)
	// Reject only the state-bearing ADK event, after the function executes.
	_, err = admin.ExecContext(t.Context(), `CREATE FUNCTION expense_agent.test_reject_state() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.data->'actions'->'stateDelta' ? 'agent_context' THEN RAISE EXCEPTION 'synthetic state failure'; END IF; RETURN NEW; END $$;
CREATE TRIGGER test_reject_state BEFORE INSERT ON expense_agent.events FOR EACH ROW EXECUTE FUNCTION expense_agent.test_reject_state()`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := admin.ExecContext(context.Background(), `DROP TRIGGER IF EXISTS test_reject_state ON expense_agent.events; DROP FUNCTION IF EXISTS expense_agent.test_reject_state()`)
		require.NoError(t, err)
	})
	outputs, _, err := runMessage(t, app, conversation, "three", "unset")
	require.Error(t, err)
	require.Equal(t, "error", outputs[len(outputs)-1].Kind)
	key, err := reopened.ResolveThread(t.Context(), sessionAppName, balance.actor, conversation, false)
	require.NoError(t, err)
	stored, err := reopened.Get(t.Context(), &session.GetRequest{AppName: sessionAppName, UserID: balance.actor, SessionID: key.Session})
	require.NoError(t, err)
	value, err := stored.Session.State().Get(updatestate.ContextKey)
	require.NoError(t, err)
	require.Contains(t, value.(map[string]any), "filters")
	history, err := app.ListRuns(t.Context(), conversation)
	require.NoError(t, err)
	found := false
	for _, run := range history {
		if run.ClientMessageID == "three" {
			found = true
			require.Equal(t, "failed", run.Status)
		}
	}
	require.True(t, found)
}

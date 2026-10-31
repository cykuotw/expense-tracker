package session

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"expense-tracker/agent/internal/databasebootstrap"
	"expense-tracker/agent/internal/models"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

func testStore(t *testing.T) (*Store, *sql.DB) {
	t.Helper()
	if os.Getenv("AGENT_TEST_POSTGRES") != "1" {
		t.Skip("requires isolated migrated PostgreSQL")
	}
	admin, err := sql.Open("pgx", os.Getenv("AGENT_STATE_ADMIN_DATABASE_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { admin.Close() })
	require.NoError(t, databasebootstrap.ProvisionRole(t.Context(), admin, "agent_state_test", "synthetic_state_password"))
	store, err := Open(t.Context(), os.Getenv("AGENT_STATE_DATABASE_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store, admin
}
func testKey() models.Key {
	return models.Key{App: "test-" + uuid.New().String(), Owner: uuid.New().String(), Session: uuid.New().String()}
}

func TestEventSessionIsolationAndRunReference(t *testing.T) {
	store, admin := testStore(t)
	key := testKey()
	run, err := store.Begin(t.Context(), key, "one", nil)
	require.NoError(t, err)
	other := key
	other.Session = uuid.New().String()
	_, err = store.Begin(t.Context(), other, "two", nil)
	require.NoError(t, err)
	runs, err := store.ListRuns(t.Context(), key)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, run.ID, runs[0].ID)
	_, err = store.ReadRunOutput(t.Context(), other, run.ID, "")
	require.ErrorIs(t, err, ErrMissing)
	for _, forged := range []models.Key{
		{App: key.App, Owner: "other-owner", Session: key.Session},
		{App: "other-app", Owner: key.Owner, Session: key.Session},
	} {
		_, err = store.ListRuns(t.Context(), forged)
		require.ErrorIs(t, err, ErrMissing)
		_, err = store.ReadRunOutput(t.Context(), forged, run.ID, "")
		require.ErrorIs(t, err, ErrMissing)
	}
	_, err = admin.ExecContext(t.Context(), `INSERT INTO expense_agent.events(id,session_id,run_id,event_type,author,data,created_at)
VALUES($1,$2,$3,'test','system','{}',clock_timestamp())`, uuid.New().String(), other.Session, run.ID)
	require.Error(t, err, "a run cannot reference an event in another session")
}

func TestUIReferenceIsolationAndSessionIdentity(t *testing.T) {
	store, _ := testStore(t)
	app, owner, thread := "ui-"+uuid.New().String(), uuid.New().String(), "browser-thread"
	key, err := store.ResolveThread(t.Context(), app, owner, thread, true)
	require.NoError(t, err)
	_, err = uuid.Parse(key.Session)
	require.NoError(t, err)
	require.NotEqual(t, thread, key.Session)
	restored, err := New(store.db).ResolveThread(t.Context(), app, owner, thread, false)
	require.NoError(t, err)
	require.Equal(t, key, restored)
	other, err := store.ResolveThread(t.Context(), app, "other", thread, true)
	require.NoError(t, err)
	require.NotEqual(t, key.Session, other.Session)
	otherApp, err := store.ResolveThread(t.Context(), app+"-other", owner, thread, true)
	require.NoError(t, err)
	require.NotEqual(t, key.Session, otherApp.Session)
	claimed, err := store.ResolveThread(t.Context(), app, owner, key.Session, true)
	require.NoError(t, err)
	require.NotEqual(t, key.Session, claimed.Session, "external IDs cannot choose an internal session")
	_, err = store.Create(t.Context(), &session.CreateRequest{AppName: app + "-other", UserID: "other", SessionID: key.Session})
	require.ErrorIs(t, err, ErrConflict, "session UUID is globally unique across owners and apps")
	ref := models.UIRef{ThreadID: thread, RunID: "browser-run"}
	run, err := store.BeginUI(t.Context(), key, ref, "one", nil)
	require.NoError(t, err)
	require.NotEqual(t, ref.RunID, run.ID)
	internalRun, err := store.ResolveRun(t.Context(), key, thread, ref.RunID)
	require.NoError(t, err)
	require.Equal(t, run.ID, internalRun)
	_, err = store.ResolveRun(t.Context(), other, thread, ref.RunID)
	require.ErrorIs(t, err, ErrMissing)
	_, err = store.ResolveRun(t.Context(), key, "other-thread", ref.RunID)
	require.ErrorIs(t, err, ErrMissing)
	require.NoError(t, store.Finish(t.Context(), run, "completed", ""))
	reused, err := New(store.db).BeginUI(t.Context(), key, ref, "two", nil)
	require.NoError(t, err)
	require.Equal(t, run.ID, reused.ID, "lookup identity survives restart without providing DB resend protection")
	ref.RunID = "browser-run-two"
	_, err = store.BeginUI(t.Context(), key, ref, "one", nil)
	require.NoError(t, err, "message resend policy belongs to the runtime, not storage")
	ref.ThreadID = "other-thread"
	_, err = store.BeginUI(t.Context(), key, ref, "two", nil)
	require.ErrorIs(t, err, ErrMissing)
	require.NoError(t, store.SoftDelete(t.Context(), key))
	_, err = store.ResolveThread(t.Context(), app, owner, thread, true)
	require.ErrorIs(t, err, ErrMissing, "retained thread mappings cannot resurrect deleted sessions")
}

func TestTimestampReplayTiesAndNoSessionCounterWrites(t *testing.T) {
	store, admin := testStore(t)
	key := testKey()
	run, err := store.Begin(t.Context(), key, "timestamps", map[string]any{"text": "hello"})
	require.NoError(t, err)
	var before, after time.Time
	require.NoError(t, admin.QueryRowContext(t.Context(), `SELECT updated_at FROM expense_agent.sessions WHERE app_name=$1`, key.App).Scan(&before))
	diagnosticID, _, err := store.Append(t.Context(), run, models.Entry{Kind: "model.response", Payload: models.Snapshot{Data: []byte(`null`)}})
	require.NoError(t, err)
	var outputs []models.Output
	for range 3 {
		require.NoError(t, store.PersistAndEmitOutput(t.Context(), run, models.Output{Kind: "text", RunID: run.ID, Delta: "visible"}, func(out models.Output) error {
			outputs = append(outputs, out)
			return nil
		}))
	}
	require.NoError(t, admin.QueryRowContext(t.Context(), `SELECT updated_at FROM expense_agent.sessions WHERE app_name=$1`, key.App).Scan(&after))
	require.True(t, before.Equal(after), "event-only writes must not update the session row")
	// Simulate imported events with identical timestamps and a clock behind the
	// retained history. Existing ties use event ID; later writes must follow them.
	var tiedAt time.Time
	require.NoError(t, admin.QueryRowContext(t.Context(), `UPDATE expense_agent.events SET created_at=statement_timestamp()+INTERVAL '1 hour' WHERE run_id=$1 AND event_type='output' RETURNING created_at`, run.ID).Scan(&tiedAt))
	for i := range outputs {
		outputs[i].CreatedAt = tiedAt.UTC()
	}
	slices.SortFunc(outputs, func(a, b models.Output) int { return strings.Compare(a.EventID, b.EventID) })
	replay, err := store.ReadRunOutput(t.Context(), key, run.ID, "")
	require.NoError(t, err)
	require.Equal(t, outputs, replay)
	page, err := store.ReadRunOutput(t.Context(), key, run.ID, outputs[0].EventID)
	require.NoError(t, err)
	require.Equal(t, outputs[1:], page)
	var next models.Output
	require.NoError(t, store.PersistAndEmitOutput(t.Context(), run, models.Output{Kind: "text", RunID: run.ID}, func(out models.Output) error { next = out; return nil }))
	require.True(t, next.CreatedAt.After(tiedAt))
	page, err = store.ReadRunOutput(t.Context(), key, run.ID, outputs[2].EventID)
	require.NoError(t, err)
	require.Equal(t, []models.Output{next}, page)
	_, err = store.ReadRunOutput(t.Context(), key, run.ID, diagnosticID)
	require.ErrorIs(t, err, ErrMissing, "diagnostic IDs cannot be used as visible replay cursors")
	_, err = store.ReadRunOutput(t.Context(), key, run.ID, uuid.New().String())
	require.ErrorIs(t, err, ErrMissing)
}

func TestSharedEventsPreserveAuthorsAndIsolateProjections(t *testing.T) {
	store, admin := testStore(t)
	key := testKey()
	run, err := store.Begin(t.Context(), key, "one", map[string]any{"text": "hello"})
	require.NoError(t, err)
	ctx := WithRun(t.Context(), store, run)
	current, err := store.Get(ctx, &session.GetRequest{AppName: key.App, UserID: key.Owner, SessionID: key.Session})
	require.NoError(t, err)
	require.NoError(t, RecordDiagnostic(ctx, "model.response", "shared-invocation", map[string]any{"agent": "first-skill", "reasoning": "diagnostic only"}))
	var nativeIDs []string
	for _, author := range []string{"user", "new-skill"} {
		event := session.NewEvent(ctx, "shared-invocation")
		// Native IDs remain payload metadata; each append stores its own record.
		event.ID = "same-native-id"
		event.Author = author
		event.Content = genai.NewContentFromText("ADK history", "user")
		event.Actions.StateDelta = map[string]any{"lastAuthor": author}
		nativeIDs = append(nativeIDs, event.ID)
		require.NoError(t, store.AppendEvent(ctx, current.Session, event))
	}
	var outputs []models.Output
	for _, kind := range []string{"started", "text"} {
		out := models.Output{Kind: kind, RunID: run.ID, ConversationID: key.Session, MessageID: run.MessageID, Skill: "new-skill", Delta: "visible"}
		require.NoError(t, store.PersistAndEmitOutput(ctx, run, out, func(stored models.Output) error { outputs = append(outputs, stored); return nil }))
	}
	// Additive dashboard metadata must not require a schema/version update or
	// change how existing history and replay readers decode payloads.
	_, err = admin.ExecContext(ctx, `UPDATE expense_agent.events SET data=data || '{"dashboardNote":"extra metadata"}'::jsonb WHERE session_id=$1`, key.Session)
	require.NoError(t, err)
	replay, err := store.ReadRunOutput(ctx, key, run.ID, "")
	require.NoError(t, err)
	require.Equal(t, outputs, replay)
	page, err := store.ReadRunOutput(ctx, key, run.ID, outputs[0].EventID)
	require.NoError(t, err)
	require.Equal(t, outputs[1:], page)
	loaded, err := store.Get(ctx, &session.GetRequest{AppName: key.App, UserID: key.Owner, SessionID: key.Session})
	require.NoError(t, err)
	require.Equal(t, 2, loaded.Session.Events().Len())
	for i, id := range nativeIDs {
		require.Equal(t, id, loaded.Session.Events().At(i).ID)
	}
	value, err := loaded.Session.State().Get("lastAuthor")
	require.NoError(t, err)
	require.Equal(t, "new-skill", value)
	rows, err := admin.QueryContext(ctx, `SELECT created_at,author,run_id::text FROM expense_agent.events WHERE session_id=$1 ORDER BY created_at,id`, key.Session)
	require.NoError(t, err)
	defer rows.Close()
	var previous time.Time
	var authors []string
	for rows.Next() {
		var createdAt time.Time
		var author, storedRun string
		require.NoError(t, rows.Scan(&createdAt, &author, &storedRun))
		require.True(t, createdAt.After(previous))
		previous = createdAt
		require.Equal(t, run.ID, storedRun)
		authors = append(authors, author)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"user", "first-skill", "user", "new-skill", "new-skill", "new-skill"}, authors)
	require.NoError(t, rows.Close())
	_, err = admin.ExecContext(ctx, `UPDATE expense_agent.events SET data='[]' WHERE session_id=$1 AND event_type IN ('adk.event','output')`, key.Session)
	require.NoError(t, err)
	_, err = store.Get(ctx, &session.GetRequest{AppName: key.App, UserID: key.Owner, SessionID: key.Session})
	require.ErrorIs(t, err, ErrStorage)
	_, err = store.ReadRunOutput(ctx, key, run.ID, "")
	require.ErrorIs(t, err, ErrStorage)

}

func TestSessionStateSignaturesAndSoftDeletion(t *testing.T) {
	store, admin := testStore(t)
	key := testKey()
	created, err := store.Create(t.Context(), &session.CreateRequest{AppName: key.App, UserID: key.Owner, SessionID: key.Session})
	require.NoError(t, err)
	signature := []byte{0, 1, 255, 64, 8}
	event := session.NewEvent(t.Context(), "invocation")
	event.Content = &genai.Content{Role: "model", Parts: []*genai.Part{{Text: "signed text", ThoughtSignature: signature}}}
	event.Actions.StateDelta = map[string]any{"pending": "balance", "step": 1, "temp:scratch": "discard", "api_key": "private-key"}
	require.NoError(t, store.AppendEvent(t.Context(), created.Session, event))
	require.Equal(t, 1, created.Session.Events().Len())
	_ = admin
	loaded, err := New(store.db).Get(t.Context(), &session.GetRequest{AppName: key.App, UserID: key.Owner, SessionID: key.Session})
	require.NoError(t, err)
	require.Equal(t, 1, loaded.Session.Events().Len())
	require.Equal(t, signature, loaded.Session.Events().At(0).Content.Parts[0].ThoughtSignature)
	stateValue, err := loaded.Session.State().Get("step")
	require.NoError(t, err)
	require.Equal(t, float64(1), stateValue)
	_, err = loaded.Session.State().Get("temp:scratch")
	require.Error(t, err)
	var latest []byte
	require.NoError(t, admin.QueryRowContext(t.Context(), `SELECT state FROM expense_agent.sessions WHERE app_name=$1 AND owner_id=$2 AND session_id=$3`, key.App, key.Owner, key.Session).Scan(&latest))
	require.Contains(t, string(latest), "pending")
	require.NotContains(t, string(latest), "private-key")
	reconstructed := map[string]any{}
	var materialized map[string]any
	require.NoError(t, json.Unmarshal(latest, &materialized))
	for event := range loaded.Session.Events().All() {
		for k, v := range event.Actions.StateDelta {
			reconstructed[k] = v
		}
	}
	require.Equal(t, materialized, reconstructed)
	_, err = store.Get(t.Context(), &session.GetRequest{AppName: key.App, UserID: "other", SessionID: key.Session})
	require.ErrorIs(t, err, session.ErrNotFound)
	require.NoError(t, store.SoftDelete(t.Context(), key))
	_, err = store.Get(t.Context(), &session.GetRequest{AppName: key.App, UserID: key.Owner, SessionID: key.Session})
	require.ErrorIs(t, err, session.ErrNotFound)
	_, err = store.Create(t.Context(), &session.CreateRequest{AppName: key.App, UserID: key.Owner, SessionID: key.Session})
	require.Error(t, err)
	require.ErrorIs(t, store.AppendEvent(t.Context(), created.Session, session.NewEvent(t.Context(), "next")), ErrMissing)
	var retained int
	require.NoError(t, admin.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM expense_agent.events WHERE session_id=$1 AND event_type='adk.event'`, key.Session).Scan(&retained))
	require.Equal(t, 1, retained)
}
func TestRunLifecycleReplayAndSingleRuntimeDeletion(t *testing.T) {
	store, _ := testStore(t)
	key := testKey()
	run, err := store.Begin(t.Context(), key, "one", map[string]any{"text": "hello"})
	require.NoError(t, err)
	output := models.Output{Kind: "started", RunID: run.ID, ConversationID: key.Session, MessageID: run.MessageID}
	var observed models.Output
	require.NoError(t, store.PersistAndEmitOutput(t.Context(), run, output, func(o models.Output) error { observed = o; return nil }))
	require.NotEmpty(t, observed.EventID)
	require.False(t, observed.CreatedAt.IsZero())
	replay, err := New(store.db).ReadRunOutput(t.Context(), key, run.ID, "")
	require.NoError(t, err)
	require.Equal(t, []models.Output{observed}, replay)
	replay, err = store.ReadRunOutput(t.Context(), key, run.ID, observed.EventID)
	require.NoError(t, err)
	require.Empty(t, replay)
	require.NoError(t, store.Finish(t.Context(), run, "completed", ""))
	runs, err := New(store.db).ListRuns(t.Context(), key)
	require.NoError(t, err)
	require.Equal(t, "completed", runs[0].Status)
	require.Equal(t, "hello", runs[0].Text)
	next, err := store.Begin(t.Context(), key, "two", nil)
	require.NoError(t, err)
	entered := make(chan struct{})
	release := make(chan struct{})
	delDone := make(chan error, 1)
	emitDone := make(chan error, 1)
	go func() {
		emitDone <- store.PersistAndEmitOutput(t.Context(), next, models.Output{Kind: "text", Delta: "bounded"}, func(models.Output) error { close(entered); <-release; return nil })
	}()
	<-entered
	go func() { delDone <- store.SoftDelete(t.Context(), key) }()
	select {
	case <-delDone:
		t.Fatal("deletion overtook active output gate")
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	require.NoError(t, <-emitDone)
	require.NoError(t, <-delDone)
	require.ErrorIs(t, store.PersistAndEmitOutput(t.Context(), next, output, func(models.Output) error { t.Fatal("output after deletion"); return nil }), ErrMissing)
	_, err = store.ReadRunOutput(t.Context(), key, next.ID, "")
	require.ErrorIs(t, err, ErrMissing)
}
func TestRoleIsolationAndAtomicFailure(t *testing.T) {
	store, admin := testStore(t)
	_, err := store.db.ExecContext(t.Context(), `SELECT * FROM public.expense LIMIT 1`)
	require.Error(t, err)
	_, err = store.db.ExecContext(t.Context(), `CREATE TABLE expense_agent.forbidden(id int)`)
	require.Error(t, err)
	require.Error(t, New(admin).VerifyRole(t.Context()))
	key := testKey()
	created, err := store.Create(t.Context(), &session.CreateRequest{AppName: key.App, UserID: key.Owner, SessionID: key.Session})
	require.NoError(t, err)
	event := session.NewEvent(t.Context(), "inv")
	event.Actions.StateDelta = map[string]any{"large": strings.Repeat("x", MaxPayloadBytes+1)}
	require.ErrorIs(t, store.AppendEvent(t.Context(), created.Session, event), ErrPayload)
	loaded, err := store.Get(t.Context(), &session.GetRequest{AppName: key.App, UserID: key.Owner, SessionID: key.Session})
	require.NoError(t, err)
	require.Zero(t, loaded.Session.Events().Len())

}

func TestEventStateWriteRollbackAndOmittedCallIdentity(t *testing.T) {
	store, admin := testStore(t)
	key := testKey()
	created, err := store.Create(t.Context(), &session.CreateRequest{AppName: key.App, UserID: key.Owner, SessionID: key.Session, State: map[string]any{"step": "initial"}})
	require.NoError(t, err)
	_, err = admin.ExecContext(t.Context(), `CREATE OR REPLACE FUNCTION expense_agent.test_reject_event() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.event_type='adk.event' THEN RAISE EXCEPTION 'synthetic event insert failure'; END IF; RETURN NEW; END $$;
 CREATE TRIGGER test_reject_event BEFORE INSERT ON expense_agent.events FOR EACH ROW EXECUTE FUNCTION expense_agent.test_reject_event()`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := admin.ExecContext(context.Background(), `DROP TRIGGER IF EXISTS test_reject_event ON expense_agent.events; DROP FUNCTION IF EXISTS expense_agent.test_reject_event()`)
		require.NoError(t, err)
	})
	event := session.NewEvent(t.Context(), "invocation")
	event.Actions.StateDelta = map[string]any{"step": "changed"}
	require.Error(t, store.AppendEvent(t.Context(), created.Session, event))
	loaded, err := store.Get(t.Context(), &session.GetRequest{AppName: key.App, UserID: key.Owner, SessionID: key.Session})
	require.NoError(t, err)
	value, err := loaded.Session.State().Get("step")
	require.NoError(t, err)
	require.Equal(t, "initial", value)
	require.Zero(t, loaded.Session.Events().Len())
	run, err := store.Begin(t.Context(), key, "one", map[string]any{"text": "hello"})
	require.NoError(t, err)
	ctx := WithRun(t.Context(), store, run)
	require.NoError(t, RecordDiagnostic(ctx, "model.request", "invocation", map[string]any{"callId": "stable-call", "large": strings.Repeat("x", MaxPayloadBytes+1)}))
	var snapshot []byte
	require.NoError(t, admin.QueryRowContext(t.Context(), `SELECT data FROM expense_agent.events WHERE run_id=$1 AND event_type='model.request'`, run.ID).Scan(&snapshot))
	var envelope models.Snapshot
	require.NoError(t, json.Unmarshal(snapshot, &envelope))
	require.Equal(t, "stable-call", envelope.CallID)
	require.False(t, envelope.Complete)
	require.Equal(t, "payload_limit", envelope.Reason)
}

func TestFailedFinalWriteLeavesIncompleteLifecycle(t *testing.T) {
	store, _ := testStore(t)
	key := testKey()
	run, err := store.Begin(t.Context(), key, "one", nil)
	require.NoError(t, err)
	require.NoError(t, store.Close())
	require.Error(t, store.Finish(t.Context(), run, "interrupted", "interrupted"))
	recovered, err := Open(t.Context(), os.Getenv("AGENT_STATE_DATABASE_URL"))
	require.NoError(t, err)
	defer recovered.Close()
	runs, err := recovered.ListRuns(t.Context(), key)
	require.NoError(t, err)
	require.Equal(t, "running", runs[0].Status, "storage reports recorded lifecycle without inferring a lease expiry")
	require.Empty(t, runs[0].ErrorCode)
}

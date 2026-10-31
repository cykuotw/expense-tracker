package integration_test

import (
	"context"
	"io"
	"testing"
	"testing/fstest"
	"time"
	"uuid"

	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/runtime"
	sessionstore "expense-tracker/agent/internal/store/session"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/session"
)

// Model/session execution is real; this fake tests the recording contract
// without requiring SQL. Output is committed before delivery is attempted.
type recordedRepository struct {
	sessionstore.Repository
	key         models.Key
	run         models.Run
	outputs     []models.Output
	reject      map[string]bool
	finalStatus string
}

func (repo *recordedRepository) ResolveThread(_ context.Context, app, owner, _ string, _ bool) (models.Key, error) {
	if repo.key.Session == "" {
		repo.key = models.Key{App: app, Owner: owner, Session: uuid.New().String()}
	}
	return repo.key, nil
}

func (repo *recordedRepository) BeginUI(_ context.Context, key models.Key, ref models.UIRef, message string, _ any) (models.Run, error) {
	repo.run = models.Run{Key: key, UI: ref, ID: uuid.New().String(), MessageID: uuid.New().String(), ClientMessageID: message, Status: "running"}
	return repo.run, nil
}

func (*recordedRepository) Append(context.Context, models.Run, models.Entry) (string, time.Time, error) {
	return uuid.New().String(), time.Now(), nil
}

func (repo *recordedRepository) PersistAndEmitOutput(ctx context.Context, _ models.Run, output models.Output, emit func(models.Output) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if repo.reject[output.Kind] {
		return sessionstore.ErrStorage
	}
	output.EventID, output.CreatedAt = uuid.New().String(), time.Now()
	repo.outputs = append(repo.outputs, output)
	return emit(output)
}

func (repo *recordedRepository) Finish(_ context.Context, _ models.Run, status, _ string) error {
	repo.finalStatus = status
	return nil
}

func newRecordedTestApp(t *testing.T, repo *recordedRepository) (*runtime.App, session.Service) {
	t.Helper()
	sessions := session.InMemoryService()
	app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
		Models: fixtures.ModelFactory, Sessions: sessions, Records: repo,
		Authenticate: func(context.Context) (models.Identity, error) {
			return models.Identity{ID: "owner", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		Skills: fstest.MapFS{
			"orchestrator/SKILL.md": {Data: []byte("---\nname: orchestrator\ndescription: Coordinate.\n---\nDelegate tasks.\n")},
			"echo/SKILL.md":         {Data: []byte("---\nname: echo\ndescription: Echo.\n---\nEcho input.\n")},
		},
	})
	require.NoError(t, err)
	return app, sessions
}

func TestRecordedRunKeepsCommittedTerminalAfterDeliveryFailure(t *testing.T) {
	repo := &recordedRepository{}
	app, sessions := newRecordedTestApp(t, repo)
	err := app.Run(t.Context(), models.ChatRequest{ConversationID: "thread", MessageID: "one", Text: "echo hello"}, func(output models.Output) error {
		if output.Kind == "finished" {
			return io.ErrClosedPipe
		}
		return nil
	})
	require.ErrorIs(t, err, io.ErrClosedPipe)
	require.Equal(t, "started", repo.outputs[0].Kind)
	require.Equal(t, "finished", repo.outputs[len(repo.outputs)-1].Kind)
	require.Empty(t, repo.finalStatus, "committed completion must not be replaced by a fallback failure")
	for _, output := range repo.outputs {
		require.NotEqual(t, "error", output.Kind)
		require.NotEmpty(t, output.MessageID)
	}
	_, err = sessions.Get(t.Context(), &session.GetRequest{AppName: sessionAppName, UserID: "owner", SessionID: repo.key.Session})
	require.NoError(t, err, "ADK must use the persisted session identity")
}

func TestRecordedRunStartFailureReleasesRetryAdmission(t *testing.T) {
	repo := &recordedRepository{reject: map[string]bool{"started": true}}
	app, _ := newRecordedTestApp(t, repo)
	request := models.ChatRequest{ConversationID: "thread", RunID: "run", MessageID: "one", Text: "echo hello"}
	err := app.Run(t.Context(), request, func(models.Output) error { return nil })
	require.ErrorIs(t, err, sessionstore.ErrStorage)
	repo.reject["started"] = false
	require.NoError(t, app.Run(t.Context(), request, func(models.Output) error { return nil }))
	require.Equal(t, "finished", repo.outputs[len(repo.outputs)-1].Kind)
	err = app.Run(t.Context(), request, func(models.Output) error { return nil })
	require.Equal(t, models.Limit, models.PublicCode(err), "a started run retains its resend guard")
}

func TestRecordedRunCancellationWritesDetachedTerminal(t *testing.T) {
	repo := &recordedRepository{}
	app, _ := newRecordedTestApp(t, repo)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	err := app.Run(ctx, models.ChatRequest{ConversationID: "thread", MessageID: "one", Text: "echo hello"}, func(output models.Output) error {
		if output.Kind == "text" {
			cancel()
			return context.Canceled
		}
		return nil
	})
	require.ErrorIs(t, err, context.Canceled)
	terminal := repo.outputs[len(repo.outputs)-1]
	require.Equal(t, "error", terminal.Kind)
	require.Equal(t, "interrupted", terminal.ErrorCode)
}

func TestRecordedRunFallsBackWhenFailureOutputCannotBeStored(t *testing.T) {
	repo := &recordedRepository{reject: map[string]bool{"text": true, "error": true}}
	app, _ := newRecordedTestApp(t, repo)
	err := app.Run(t.Context(), models.ChatRequest{ConversationID: "thread", MessageID: "one", Text: "echo hello"}, func(models.Output) error { return nil })
	require.ErrorIs(t, err, sessionstore.ErrStorage)
	require.Equal(t, "failed", repo.finalStatus)
}

func TestRecordedStartDeliveryFailureRetainsAdmission(t *testing.T) {
	repo := &recordedRepository{}
	app, _ := newRecordedTestApp(t, repo)
	request := models.ChatRequest{ConversationID: "thread", RunID: "run", MessageID: "one", Text: "echo hello"}
	deliveries := 0
	err := app.Run(t.Context(), request, func(output models.Output) error {
		deliveries++
		require.Equal(t, "started", output.Kind)
		return io.ErrClosedPipe
	})
	require.ErrorIs(t, err, io.ErrClosedPipe)
	require.Equal(t, 1, deliveries, "cleanup must not retry delivery after a disconnected stream")
	require.Len(t, repo.outputs, 2)
	require.Equal(t, "started", repo.outputs[0].Kind)
	require.Equal(t, "error", repo.outputs[1].Kind)
	err = app.Run(t.Context(), request, func(models.Output) error { return nil })
	require.Equal(t, models.Limit, models.PublicCode(err), "committed start retains its resend guard even if delivery failed")
}

func TestRecordedFailureDeliveryKeepsCommittedTerminal(t *testing.T) {
	repo := &recordedRepository{reject: map[string]bool{"text": true}}
	app, _ := newRecordedTestApp(t, repo)
	errorDeliveries := 0
	err := app.Run(t.Context(), models.ChatRequest{ConversationID: "thread", MessageID: "one", Text: "echo hello"}, func(output models.Output) error {
		if output.Kind == "error" {
			errorDeliveries++
			return io.ErrClosedPipe
		}
		return nil
	})
	require.ErrorIs(t, err, sessionstore.ErrStorage, "cleanup delivery does not replace the execution failure")
	require.Equal(t, 1, errorDeliveries)
	require.Equal(t, "error", repo.outputs[len(repo.outputs)-1].Kind)
	require.Empty(t, repo.finalStatus, "a committed error event needs no fallback status write")
}

// Simulate a row still marked running to check process-local admission status.
func (repo *recordedRepository) ListRuns(_ context.Context, key models.Key) ([]models.Run, error) {
	if key != repo.key {
		return nil, sessionstore.ErrMissing
	}
	return []models.Run{repo.run}, nil
}

func TestConversationHistoryUsesExecutionRunStatus(t *testing.T) {
	repo := &recordedRepository{}
	app, _ := newRecordedTestApp(t, repo)
	started := false
	err := app.Run(t.Context(), models.ChatRequest{ConversationID: "thread", RunID: "browser-run", MessageID: "one", Text: "echo hello"}, func(output models.Output) error {
		if output.Kind != "started" {
			return nil
		}
		started = true
		runs, err := app.ListRuns(t.Context(), "thread")
		require.NoError(t, err)
		require.Len(t, runs, 1)
		require.Equal(t, "browser-run", runs[0].ID)
		require.Equal(t, "running", runs[0].Status)
		return nil
	})
	require.NoError(t, err)
	require.True(t, started)
	runs, err := app.ListRuns(t.Context(), "thread")
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, "unknown", runs[0].Status, "a stale recorded status cannot count as an active execution")
}

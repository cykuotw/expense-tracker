package conversation

import (
	"context"
	"slices"
	"testing"
	"time"

	"expense-tracker/agent/internal/models"
	sessionstore "expense-tracker/agent/internal/store/session"

	"github.com/stretchr/testify/require"
)

// Only runtime storage and identity are available; history has no domain client.
type historyRepository struct {
	sessionstore.Repository
	key     models.Key
	run     models.Run
	outputs []models.Output
}

func (repo *historyRepository) ListConversations(_ context.Context, app, owner string) ([]models.Conversation, error) {
	if app != repo.key.App || owner != repo.key.Owner {
		return []models.Conversation{}, nil
	}
	return []models.Conversation{{ID: repo.run.UI.ThreadID}}, nil
}

func (repo *historyRepository) ResolveThread(_ context.Context, app, owner, thread string, _ bool) (models.Key, error) {
	if app != repo.key.App || owner != repo.key.Owner || thread != repo.run.UI.ThreadID {
		return models.Key{}, sessionstore.ErrMissing
	}
	return repo.key, nil
}

func (repo *historyRepository) ListRuns(_ context.Context, key models.Key) ([]models.Run, error) {
	if key != repo.key {
		return nil, sessionstore.ErrMissing
	}
	return []models.Run{repo.run}, nil
}

func (repo *historyRepository) ResolveRun(_ context.Context, key models.Key, thread, run string) (string, error) {
	if key != repo.key || thread != repo.run.UI.ThreadID || run != repo.run.UI.RunID {
		return "", sessionstore.ErrMissing
	}
	return repo.run.ID, nil
}

func (repo *historyRepository) ReadRunOutput(_ context.Context, key models.Key, run, _ string) ([]models.Output, error) {
	if key != repo.key || run != repo.run.ID {
		return nil, sessionstore.ErrMissing
	}
	return slices.Clone(repo.outputs), nil
}

func TestHistoryRequiresOnlyIdentityAndOwnedRuntimeRecords(t *testing.T) {
	key := models.Key{App: "expense-agent", Owner: "owner", Session: "internal-session"}
	repo := &historyRepository{
		key:     key,
		run:     models.Run{Key: key, ID: "internal-run", UI: models.UIRef{ThreadID: "conversation", RunID: "browser-run"}, Status: "completed"},
		outputs: []models.Output{{Kind: "text", Delta: "Historical tool result"}},
	}
	identity := models.Identity{ID: key.Owner, ExpiresAt: time.Now().Add(time.Hour)}
	app := New("expense-agent", repo, func(context.Context) (string, error) {
		if !identity.ExpiresAt.After(time.Now()) {
			return "", &models.Error{Code: models.Unauthenticated}
		}
		return identity.ID, nil
	}, nil)
	items, err := app.ListConversations(t.Context())
	require.NoError(t, err)
	require.Equal(t, []models.Conversation{{ID: "conversation"}}, items)
	runs, err := app.ListRuns(t.Context(), "conversation")
	require.NoError(t, err)
	require.Equal(t, "browser-run", runs[0].ID)
	output, err := app.ReadRunOutput(t.Context(), "conversation", "browser-run", "")
	require.NoError(t, err)
	require.Equal(t, repo.outputs, output)
	identity.ID = "other-owner"
	items, err = app.ListConversations(t.Context())
	require.NoError(t, err)
	require.Empty(t, items)
	_, err = app.ReadRunOutput(t.Context(), "conversation", "browser-run", "")
	require.ErrorIs(t, err, sessionstore.ErrMissing)
	identity.ExpiresAt = time.Now().Add(-time.Minute)
	_, err = app.ListRuns(t.Context(), "conversation")
	require.Equal(t, models.Unauthenticated, models.PublicCode(err))
}

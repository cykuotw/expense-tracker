package group

import (
	"database/sql/driver"
	"errors"
	"expense-tracker/backend/internal/testsql"
	"expense-tracker/backend/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGroupLifecycleCapabilities(t *testing.T) {
	for _, test := range []struct {
		name                                       string
		active, creator, settled, archive, restore bool
	}{
		{"empty settled creator", true, true, true, true, false},
		{"unsettled creator", true, true, false, false, false},
		{"archived creator", false, true, true, false, true},
		{"settled member", true, false, true, false, false},
		{"archived member", false, false, true, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, cleanup := testsql.Open(testsql.Result{Columns: []string{"active", "creator", "settled"}, Rows: [][]driver.Value{{test.active, test.creator, test.settled}}})
			t.Cleanup(cleanup)
			state, err := NewStore(db).GetGroupLifecycle(uuid.NewString(), uuid.NewString())
			require.NoError(t, err)
			require.Equal(t, test.archive, state.CanArchive)
			require.Equal(t, test.restore, state.CanRestore)
			require.Equal(t, test.creator, state.CanManageLifecycle)
			require.Equal(t, test.active && !test.settled, state.ArchiveBlockedReason != "")
		})
	}
}

func TestGroupLifecycleReadFailures(t *testing.T) {
	for _, test := range []struct {
		name   string
		result testsql.Result
		want   error
	}{
		{"unauthorized or absent", testsql.Result{Columns: []string{"active", "creator", "settled"}}, types.ErrGroupNotExist},
		{"database failure", testsql.Result{QueryErr: types.ErrInvalidAction}, types.ErrInvalidAction},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, cleanup := testsql.Open(test.result)
			t.Cleanup(cleanup)
			_, err := NewStore(db).GetGroupLifecycle(uuid.NewString(), uuid.NewString())
			require.ErrorIs(t, err, test.want)
		})
	}
}

func TestArchivedGroupCursorAndBounds(t *testing.T) {
	db, cleanup := testsql.Open()
	t.Cleanup(cleanup)
	store := NewStore(db)
	for _, cursor := range []string{"not-base64!", "e30", "", "eyJpZCI6Im5vdC1hLXV1aWQifQ"} {
		_, err := store.GetArchivedGroups(uuid.NewString(), cursor, 0)
		require.ErrorIs(t, err, types.ErrInvalidGroupCursor)
		if cursor != "" {
			_, err = store.GetArchivedGroups(uuid.NewString(), cursor, 20)
			require.ErrorIs(t, err, types.ErrInvalidGroupCursor)
		}
	}
	_, err := store.GetArchivedGroups(uuid.NewString(), "", 101)
	require.ErrorIs(t, err, types.ErrInvalidGroupCursor)
}

func TestArchivedGroupPaginationPreservesLastReturnedPosition(t *testing.T) {
	created := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	id := uuid.New()
	db, cleanup := testsql.Open(testsql.Result{
		Columns: []string{"id", "name", "description", "currency", "type", "created"},
		Rows:    [][]driver.Value{{id.String(), "First", "", "CAD", "trip", created}, {uuid.NewString(), "Second", "", "CAD", "trip", created}},
	})
	t.Cleanup(cleanup)
	page, err := NewStore(db).GetArchivedGroups(uuid.NewString(), "", 1)
	require.NoError(t, err)
	require.Len(t, page.Groups, 1)
	position, err := decodeArchivedCursor(page.NextCursor)
	require.NoError(t, err)
	require.Equal(t, id, position.ID)
	require.True(t, position.Created.Equal(created))
}

func TestArchivedGroupPaginationPropagatesRowsError(t *testing.T) {
	want := errors.New("iteration failed")
	db, cleanup := testsql.Open(testsql.Result{Columns: []string{"id", "name", "description", "currency", "type", "created"}, IterationErr: want})
	t.Cleanup(cleanup)
	_, err := NewStore(db).GetArchivedGroups(uuid.NewString(), "", 20)
	require.ErrorIs(t, err, want)
}

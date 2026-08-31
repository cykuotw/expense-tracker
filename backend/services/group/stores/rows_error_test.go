package group

import (
	"database/sql/driver"
	"errors"
	"expense-tracker/backend/internal/testsql"
	"expense-tracker/backend/types"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetGroupCurrencyReturnsQueryAndScanErrors(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		wantErr := errors.New("currency query failed")
		db, cleanup := testsql.Open(testsql.Result{QueryErr: wantErr})
		t.Cleanup(cleanup)

		currency, err := NewStore(db).GetGroupCurrency("group-id")

		require.Empty(t, currency)
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("scan", func(t *testing.T) {
		db, cleanup := testsql.Open(testsql.Result{
			Columns: []string{"currency", "unexpected"},
			Rows:    [][]driver.Value{{"CAD", "value"}},
		})
		t.Cleanup(cleanup)

		currency, err := NewStore(db).GetGroupCurrency("group-id")

		require.Empty(t, currency)
		require.Error(t, err)
	})
}

func TestGetGroupCurrencyReturnsRowsErrorBeforeNotFound(t *testing.T) {
	wantErr := errors.New("currency rows iteration failed")
	db, cleanup := testsql.Open(testsql.Result{
		Columns:      []string{"currency"},
		IterationErr: wantErr,
	})
	t.Cleanup(cleanup)

	currency, err := NewStore(db).GetGroupCurrency("group-id")

	require.Empty(t, currency)
	require.ErrorIs(t, err, wantErr)
	require.NotErrorIs(t, err, types.ErrGroupNotExist)
}

func TestGetGroupByIDAndUserPreservesGetGroupDatabaseError(t *testing.T) {
	wantErr := errors.New("group read failed")
	db, cleanup := testsql.Open(
		boolResult(true),
		boolResult(true),
		boolResult(true),
		testsql.Result{QueryErr: wantErr},
	)
	t.Cleanup(cleanup)

	group, err := NewStore(db).GetGroupByIDAndUser("group-id", "user-id")

	require.Nil(t, group)
	require.ErrorIs(t, err, wantErr)
	require.NotErrorIs(t, err, types.ErrGroupNotExist)
}

func TestGetGroupMemberClosesInnerRows(t *testing.T) {
	wantIterationErr := errors.New("member rows iteration failed")
	testCases := []struct {
		name       string
		inner      testsql.Result
		wantError  bool
		wantResult int
	}{
		{
			name: "success",
			inner: testsql.Result{
				Columns: groupMemberUserColumns(),
				Rows: [][]driver.Value{{
					uuid.NewString(), "username", "first", "last", "user@example.com", "hash",
					"", "", time.Now().UTC(), true, "nickname", "user",
				}},
			},
			wantResult: 1,
		},
		{
			name: "scan error",
			inner: testsql.Result{
				Columns: []string{"id"},
				Rows:    [][]driver.Value{{"user-id"}},
			},
			wantError: true,
		},
		{
			name: "iteration error",
			inner: testsql.Result{
				Columns:      groupMemberUserColumns(),
				IterationErr: wantIterationErr,
			},
			wantError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			var innerClosed atomic.Bool
			test.inner.OnRowsClose = func() { innerClosed.Store(true) }
			db, cleanup := testsql.Open(test.inner)
			t.Cleanup(cleanup)

			users, err := NewStore(db).GetGroupMemberByGroupID("group-id")

			if test.wantError {
				require.Nil(t, users)
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, users, test.wantResult)
			}
			require.True(t, innerClosed.Load())
		})
	}
}

func boolResult(value bool) testsql.Result {
	return testsql.Result{
		Columns: []string{"exists"},
		Rows:    [][]driver.Value{{value}},
	}
}

func groupMemberUserColumns() []string {
	return []string{
		"id", "username", "firstname", "lastname", "email", "password_hash",
		"external_type", "external_id", "create_time_utc", "is_active", "nickname", "role",
	}
}

package user

import (
	"errors"
	"expense-tracker/backend/internal/testsql"
	"expense-tracker/backend/types"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckUserExistReturnsQueryError(t *testing.T) {
	wantErr := errors.New("user query failed")
	db, cleanup := testsql.Open(testsql.Result{QueryErr: wantErr})
	t.Cleanup(cleanup)

	exists, err := NewStore(db).CheckUserExistByID("user-id")

	require.False(t, exists)
	require.ErrorIs(t, err, wantErr)
}

func TestGetUserByEmailReturnsRowsErrorBeforeNotFound(t *testing.T) {
	wantErr := errors.New("user rows iteration failed")
	db, cleanup := testsql.Open(testsql.Result{
		Columns:      []string{"id", "username", "firstname", "lastname", "email", "password_hash", "external_type", "external_id", "create_time_utc", "is_active", "nickname", "role"},
		IterationErr: wantErr,
	})
	t.Cleanup(cleanup)

	result, err := NewStore(db).GetUserByEmail("user@example.com")

	require.Nil(t, result)
	require.ErrorIs(t, err, wantErr)
	require.NotErrorIs(t, err, types.ErrUserNotExist)
}

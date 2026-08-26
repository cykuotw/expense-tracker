package invitation

import (
	"database/sql/driver"
	"errors"
	"expense-tracker/backend/internal/testsql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetInvitationByTokenReturnsRowsErrorBeforeNotFound(t *testing.T) {
	wantErr := errors.New("invitation rows iteration failed")
	db, cleanup := testsql.Open(testsql.Result{
		Columns:      invitationColumns(),
		IterationErr: wantErr,
	})
	t.Cleanup(cleanup)

	result, err := NewStore(db).GetInvitationByToken("token")

	require.Nil(t, result)
	require.ErrorIs(t, err, wantErr)
}

func TestGetInvitationsDoesNotReturnPartialRows(t *testing.T) {
	wantErr := errors.New("invitation rows iteration failed")
	now := time.Now().UTC()
	db, cleanup := testsql.Open(testsql.Result{
		Columns: invitationColumns(),
		Rows: [][]driver.Value{{
			uuid.NewString(), "token", "invitee@example.com", uuid.NewString(), now, nil, now,
		}},
		IterationErr: wantErr,
	})
	t.Cleanup(cleanup)

	result, err := NewStore(db).GetInvitations()

	require.Nil(t, result)
	require.ErrorIs(t, err, wantErr)
}

func invitationColumns() []string {
	return []string{"id", "token", "email", "inviter_id", "expires_at", "used_at", "created_at"}
}

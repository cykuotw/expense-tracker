package store

import (
	"database/sql/driver"
	"errors"
	"expense-tracker/backend/internal/testsql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetExpenseTypeReturnsScanError(t *testing.T) {
	db, cleanup := testsql.Open(testsql.Result{
		Columns: []string{"id", "name"},
		Rows:    [][]driver.Value{{uuid.NewString(), "Dining"}},
	})
	t.Cleanup(cleanup)

	result, err := NewStore(db).GetExpenseType()

	require.Nil(t, result)
	require.Error(t, err)
}

func TestGetExpenseTypeDoesNotReturnPartialRows(t *testing.T) {
	wantErr := errors.New("expense type rows iteration failed")
	db, cleanup := testsql.Open(testsql.Result{
		Columns: []string{"id", "name", "category"},
		Rows: [][]driver.Value{{
			uuid.NewString(), "Restaurant", "Dining",
		}},
		IterationErr: wantErr,
	})
	t.Cleanup(cleanup)

	result, err := NewStore(db).GetExpenseType()

	require.Nil(t, result)
	require.ErrorIs(t, err, wantErr)
}

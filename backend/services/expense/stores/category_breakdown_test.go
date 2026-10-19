package store

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"expense-tracker/backend/internal/testsql"
	"expense-tracker/backend/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"testing"
)

type categoryTransaction struct {
	fakeTransaction
	inserts     int
	failSecond  bool
	missingType bool
}

func (tx *categoryTransaction) Exec(query string, args ...any) (sql.Result, error) {
	if len(args) > 1 {
		tx.inserts++
		if tx.failSecond && tx.inserts == 2 {
			return nil, errors.New("second category failed")
		}
		if tx.missingType {
			return driver.RowsAffected(0), nil
		}
	}
	return driver.RowsAffected(1), nil
}
func TestCategoryReplacementRollsBackPartialWrite(t *testing.T) {
	rows := []types.ExpenseCategoryAmount{{ExpenseTypeID: uuid.New(), Amount: decimal.NewFromInt(200)}, {ExpenseTypeID: uuid.New(), Amount: decimal.NewFromInt(100)}}
	for _, fail := range []bool{false, true} {
		tx := &categoryTransaction{failSecond: fail}
		store := &Store{beginTx: func() (transaction, error) { return tx, nil }}
		err := store.RunInTransaction(func(s types.ExpenseTransactionStore) error {
			return s.ReplaceExpenseCategoryBreakdown(uuid.New(), rows)
		})
		if fail {
			require.ErrorContains(t, err, "second category failed")
			require.Equal(t, 1, tx.rollbackCalls)
			require.Zero(t, tx.commitCalls)
		} else {
			require.NoError(t, err)
			require.Equal(t, 1, tx.commitCalls)
			require.Zero(t, tx.rollbackCalls)
		}
	}
	tx := &categoryTransaction{missingType: true}
	store := &Store{db: tx}
	require.ErrorIs(t, store.ReplaceExpenseCategoryBreakdown(uuid.New(), rows), types.ErrInvalidCategoryBreakdown)
}
func TestCategoryReadDoesNotReturnPartialRows(t *testing.T) {
	db, cleanup := testsql.Open(testsql.Result{Columns: []string{"expense_id", "expense_type_id", "amount", "note", "name", "category"}, Rows: [][]driver.Value{{uuid.NewString(), uuid.NewString(), "200", "Food", "Groceries", "Food and Drink"}}, IterationErr: errors.New("read failed")})
	t.Cleanup(cleanup)
	result, err := NewStore(db).GetCategoryBreakdownsByExpenseIDs([]string{uuid.NewString()})
	require.ErrorContains(t, err, "read failed")
	require.Nil(t, result)
}

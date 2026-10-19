package store_test

import (
	store "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCategoryBreakdownPersistenceLifecycle(t *testing.T) {
	database := openTestDB(t)
	expense := newTestExpense(uuid.New())
	expense.Total = decimal.NewFromInt(300)
	require.NoError(t, ensureExpenseParents(database, expense))
	household := uuid.New()
	_, err := database.Exec(`INSERT INTO expense_type(id,name,category) VALUES ($1,$2,'Home')`, household, "Household-"+household.String()[:8])
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := database.Exec(`DELETE FROM expense WHERE id = $1`, expense.ID)
		require.NoError(t, err)
		_, err = database.Exec(`DELETE FROM expense_type WHERE id = $1`, household)
		require.NoError(t, err)
	})
	s := store.NewStore(database)
	rows := []types.ExpenseCategoryAmount{{ExpenseTypeID: expense.ExpenseTypeID, Amount: decimal.NewFromInt(200), Note: "Groceries"}, {ExpenseTypeID: household, Amount: decimal.NewFromInt(100), Note: "Trolley"}}
	require.NoError(t, s.RunInTransaction(func(tx types.ExpenseTransactionStore) error {
		if err := tx.CreateExpense(expense); err != nil {
			return err
		}
		return tx.ReplaceExpenseCategoryBreakdown(expense.ID, rows)
	}))
	read := func() []types.ExpenseCategoryAmount {
		t.Helper()
		result, err := s.GetCategoryBreakdownsByExpenseIDs([]string{expense.ID.String()})
		require.NoError(t, err)
		return result[expense.ID.String()]
	}
	stored := read()
	require.Len(t, stored, 2)
	require.Equal(t, "Trolley", stored[1].Note)
	require.Equal(t, "Home", stored[1].Category)
	// A bad second type must restore the original rows after the first insert.
	broken := append([]types.ExpenseCategoryAmount(nil), rows...)
	broken[1].ExpenseTypeID = uuid.New()
	require.ErrorIs(t, s.RunInTransaction(func(tx types.ExpenseTransactionStore) error {
		return tx.ReplaceExpenseCategoryBreakdown(expense.ID, broken)
	}), types.ErrInvalidCategoryBreakdown)
	require.Equal(t, stored, read())
	require.NoError(t, s.RunInTransaction(func(tx types.ExpenseTransactionStore) error {
		return tx.ReplaceExpenseCategoryBreakdown(expense.ID, nil)
	}))
	require.Empty(t, read())
	var effective string
	require.NoError(t, database.QueryRow(`SELECT amount::text FROM expense_effective_category WHERE expense_id = $1`, expense.ID).Scan(&effective))
	require.True(t, decimal.RequireFromString(effective).Equal(expense.Total))
	require.NoError(t, s.RunInTransaction(func(tx types.ExpenseTransactionStore) error {
		return tx.ReplaceExpenseCategoryBreakdown(expense.ID, rows)
	}))
	_, err = database.Exec(`UPDATE expense SET is_deleted = TRUE WHERE id = $1`, expense.ID)
	require.NoError(t, err)
	require.Len(t, read(), 2)
	_, err = database.Exec(`DELETE FROM expense WHERE id = $1`, expense.ID)
	require.NoError(t, err)
	require.Empty(t, read())
}

package store_test

import (
	"testing"

	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpenseAllocationPersistenceAndLedgerReconciliation(t *testing.T) {
	db := openTestDB(t)
	store := expense.NewStore(db)
	expenseRecord := newTestExpense(uuid.New())
	firstBorrowerID := uuid.New()
	secondBorrowerID := uuid.New()
	replacementBorrowerID := uuid.New()
	newPayerID := uuid.New()
	require.NoError(t, insertExpense(db, expenseRecord))
	defer deleteExpense(db, expenseRecord.ID)
	for _, userID := range []uuid.UUID{firstBorrowerID, secondBorrowerID, replacementBorrowerID, newPayerID} {
		require.NoError(t, ensureUser(db, userID))
		require.NoError(t, ensureMember(db, expenseRecord.GroupID, userID))
	}

	firstAmount := decimal.RequireFromString("4.00")
	secondAmount := decimal.RequireFromString("6.00")
	require.NoError(t, store.CreateExpenseAllocation(types.ExpenseAllocation{
		ExpenseID: expenseRecord.ID, UserID: firstBorrowerID, Amount: &firstAmount,
	}))
	require.NoError(t, store.CreateExpenseAllocation(types.ExpenseAllocation{
		ExpenseID: expenseRecord.ID, UserID: secondBorrowerID, Amount: &secondAmount,
	}))

	stored, err := store.GetExpenseAllocationsByExpenseID(expenseRecord.ID.String())
	require.NoError(t, err)
	require.Len(t, stored, 2)
	storedByUser := make(map[uuid.UUID]types.ExpenseAllocation, len(stored))
	for _, allocation := range stored {
		storedByUser[allocation.UserID] = allocation
	}
	assert.True(t, storedByUser[firstBorrowerID].Amount.Equal(firstAmount))
	assert.True(t, storedByUser[secondBorrowerID].Amount.Equal(secondAmount))

	retainedLedgerID := uuid.New()
	require.NoError(t, store.CreateLedger(types.Ledger{
		ID: retainedLedgerID, ExpenseID: expenseRecord.ID,
		LenderUserID: expenseRecord.PayByUserId, BorrowerUesrID: firstBorrowerID,
		Share: firstAmount,
	}))
	require.NoError(t, store.CreateLedger(types.Ledger{
		ID: uuid.New(), ExpenseID: expenseRecord.ID,
		LenderUserID: expenseRecord.PayByUserId, BorrowerUesrID: secondBorrowerID,
		Share: secondAmount,
	}))

	firstReplacementAmount := decimal.RequireFromString("3.00")
	secondReplacementAmount := decimal.RequireFromString("7.00")
	replacementAllocations := []types.ExpenseAllocation{
		{ExpenseID: expenseRecord.ID, UserID: firstBorrowerID, Amount: &firstReplacementAmount},
		{ExpenseID: expenseRecord.ID, UserID: replacementBorrowerID, Amount: &secondReplacementAmount},
	}
	replacementLedgers := []types.Ledger{
		{BorrowerUesrID: firstBorrowerID, Share: decimal.RequireFromString("3.00")},
		{BorrowerUesrID: replacementBorrowerID, Share: decimal.RequireFromString("7.00")},
	}
	require.NoError(t, store.RunInTransaction(func(tx types.ExpenseTransactionStore) error {
		return tx.ReconcileExpenseAllocationState(
			expenseRecord.ID,
			newPayerID,
			replacementAllocations,
			replacementLedgers,
		)
	}))

	rows, err := db.Query(`SELECT id, lender_user_id, borrower_user_id, share
		FROM ledger WHERE expense_id = $1 ORDER BY borrower_user_id`, expenseRecord.ID)
	require.NoError(t, err)
	defer rows.Close()
	type storedLedger struct {
		id, lenderID, borrowerID uuid.UUID
		share                    decimal.Decimal
	}
	ledgers := make([]storedLedger, 0, 2)
	for rows.Next() {
		var ledger storedLedger
		require.NoError(t, rows.Scan(&ledger.id, &ledger.lenderID, &ledger.borrowerID, &ledger.share))
		ledgers = append(ledgers, ledger)
	}
	require.NoError(t, rows.Err())
	require.Len(t, ledgers, 2)

	byBorrower := make(map[uuid.UUID]storedLedger, len(ledgers))
	for _, ledger := range ledgers {
		byBorrower[ledger.borrowerID] = ledger
		assert.Equal(t, newPayerID, ledger.lenderID)
	}
	assert.Equal(t, retainedLedgerID, byBorrower[firstBorrowerID].id)
	assert.True(t, byBorrower[firstBorrowerID].share.Equal(decimal.RequireFromString("3.00")))
	assert.True(t, byBorrower[replacementBorrowerID].share.Equal(decimal.RequireFromString("7.00")))
	_, staleExists := byBorrower[secondBorrowerID]
	assert.False(t, staleExists)
}

func TestReplaceExpenseAllocationsRetainsSelectedZeroValueParticipant(t *testing.T) {
	db := openTestDB(t)
	store := expense.NewStore(db)
	expenseRecord := newTestExpense(uuid.New())
	participantID := uuid.New()
	require.NoError(t, ensureUser(db, participantID))
	require.NoError(t, insertExpense(db, expenseRecord))
	defer deleteExpense(db, expenseRecord.ID)

	zero := decimal.Zero
	allocations := []types.ExpenseAllocation{{
		ExpenseID: expenseRecord.ID, UserID: participantID, Amount: &zero,
	}}
	require.Error(t, store.ReconcileExpenseAllocationState(
		expenseRecord.ID, expenseRecord.PayByUserId, allocations, nil,
	), "the root store must not allow destructive reconciliation outside a transaction")
	require.NoError(t, store.RunInTransaction(func(tx types.ExpenseTransactionStore) error {
		return tx.ReconcileExpenseAllocationState(
			expenseRecord.ID, expenseRecord.PayByUserId, allocations, nil,
		)
	}))

	stored, err := store.GetExpenseAllocationsByExpenseID(expenseRecord.ID.String())
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, participantID, stored[0].UserID)
	assert.True(t, stored[0].Amount.Equal(decimal.Zero))

	err = store.RunInTransaction(func(tx types.ExpenseTransactionStore) error {
		return tx.ReconcileExpenseAllocationState(
			expenseRecord.ID,
			expenseRecord.PayByUserId,
			[]types.ExpenseAllocation{{
				ExpenseID: uuid.New(), UserID: participantID, Amount: &zero,
			}},
			nil,
		)
	})
	require.ErrorIs(t, err, types.ErrInvalidAction)
	stored, err = store.GetExpenseAllocationsByExpenseID(expenseRecord.ID.String())
	require.NoError(t, err)
	require.Len(t, stored, 1, "a rejected cross-expense row must not delete existing allocation data")
}

package store_test

import (
	"errors"
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettlementTransactionRollsBackExpenseAndBalanceMutations(t *testing.T) {
	db := openTestDB(t)
	store := expense.NewStore(db)
	groupID := uuid.New()
	creatorID := uuid.New()
	counterpartyID := uuid.New()
	expenseTypeID := uuid.New()
	expenseID := uuid.New()
	balance := &types.Balance{
		ID:             uuid.New(),
		SenderUserID:   counterpartyID,
		ReceiverUserID: creatorID,
		Share:          decimal.NewFromInt(10),
		GroupID:        groupID,
	}
	expenseRecord := types.Expense{
		ID:             expenseID,
		Description:    "settlement rollback",
		GroupID:        groupID,
		CreateByUserID: creatorID,
		PayByUserId:    creatorID,
		ExpenseTypeID:  expenseTypeID,
		Total:          decimal.NewFromInt(10),
		Currency:       "CAD",
		AllocationMode: types.ExpenseAllocationEqual,
	}
	t.Cleanup(func() {
		deleteBalances(db, []*types.Balance{balance})
		deleteExpense(db, expenseID)
		_, err := db.Exec("DELETE FROM groups WHERE id = $1", groupID)
		require.NoError(t, err)
		_, err = db.Exec("DELETE FROM expense_type WHERE id = $1", expenseTypeID)
		require.NoError(t, err)
		_, err = db.Exec("DELETE FROM users WHERE id = ANY($1)", []uuid.UUID{creatorID, counterpartyID})
		require.NoError(t, err)
	})

	require.NoError(t, insertExpense(db, expenseRecord))
	require.NoError(t, insertBalance(db, balance))
	injectedErr := errors.New("fail after settlement writes")

	err := store.RunInTransaction(func(transactionStore types.ExpenseTransactionStore) error {
		if _, err := transactionStore.LockGroupCurrency(groupID.String()); err != nil {
			return err
		}
		if err := transactionStore.UpdateExpenseSettleInGroup(groupID.String()); err != nil {
			return err
		}
		if err := transactionStore.OutdateBalanceByGroupId(groupID.String()); err != nil {
			return err
		}
		return injectedErr
	})
	require.ErrorIs(t, err, injectedErr)

	assert.False(t, selectExpenseByID(db, expenseID).IsSettled)
	assert.False(t, selectBalance(db, balance.ID).IsOutdated)
}

func TestActiveGroupLockSerializesAccountingTransactions(t *testing.T) {
	db := openTestDB(t)
	store := expense.NewStore(db)
	groupID := uuid.New()
	creatorID := uuid.New()
	require.NoError(t, ensureUser(db, creatorID))
	require.NoError(t, ensureGroup(db, groupID, creatorID))
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups WHERE id = $1", groupID)
		require.NoError(t, err)
		_, err = db.Exec("DELETE FROM users WHERE id = $1", creatorID)
		require.NoError(t, err)
	})

	firstLocked := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- store.RunInTransaction(func(transactionStore types.ExpenseTransactionStore) error {
			if _, err := transactionStore.LockGroupCurrency(groupID.String()); err != nil {
				return err
			}
			close(firstLocked)
			<-releaseFirst
			return nil
		})
	}()
	<-firstLocked

	secondAcquired := make(chan struct{})
	secondStarted := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		close(secondStarted)
		secondDone <- store.RunInTransaction(func(transactionStore types.ExpenseTransactionStore) error {
			if _, err := transactionStore.LockGroupCurrency(groupID.String()); err != nil {
				return err
			}
			close(secondAcquired)
			return nil
		})
	}()
	<-secondStarted

	blocked := false
	select {
	case <-secondAcquired:
	case <-time.After(100 * time.Millisecond):
		blocked = true
	}
	close(releaseFirst)

	assert.True(t, blocked, "the second accounting transaction must wait for the group lock")
	require.NoError(t, <-firstDone)
	require.NoError(t, <-secondDone)
}

func TestActiveGroupLockRejectsArchivedGroup(t *testing.T) {
	db := openTestDB(t)
	store := expense.NewStore(db)
	groupID := uuid.New()
	creatorID := uuid.New()
	require.NoError(t, ensureUser(db, creatorID))
	require.NoError(t, ensureGroup(db, groupID, creatorID))
	_, err := db.Exec("UPDATE groups SET is_active = FALSE WHERE id = $1", groupID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := db.Exec("DELETE FROM groups WHERE id = $1", groupID)
		require.NoError(t, cleanupErr)
		_, cleanupErr = db.Exec("DELETE FROM users WHERE id = $1", creatorID)
		require.NoError(t, cleanupErr)
	})

	err = store.RunInTransaction(func(transactionStore types.ExpenseTransactionStore) error {
		_, err := transactionStore.LockGroupCurrency(groupID.String())
		return err
	})

	require.ErrorIs(t, err, types.ErrGroupNotExist)
}

func TestRunInTransactionRollsBackMidTransactionChildFailure(t *testing.T) {
	db := openTestDB(t)
	store := expense.NewStore(db)
	expenseID := uuid.New()
	itemID := uuid.New()
	expense := types.Expense{
		ID:             expenseID,
		Description:    "transaction rollback test",
		GroupID:        uuid.New(),
		CreateByUserID: uuid.New(),
		PayByUserId:    uuid.New(),
		ExpenseTypeID:  uuid.New(),
		SubTotal:       decimal.NewFromInt(10),
		TaxFeeTip:      decimal.Zero,
		Total:          decimal.NewFromInt(10),
		Currency:       "CAD",
		AllocationMode: "equal",
	}
	item := types.Item{
		ID:        itemID,
		ExpenseID: expenseID,
		Name:      "duplicate item",
		Amount:    decimal.NewFromInt(1),
		Unit:      "each",
		UnitPrice: decimal.NewFromInt(10),
	}
	require.NoError(t, ensureExpenseParents(db, expense))

	defer deleteExpense(db, expenseID)
	defer deleteItem(db, itemID)

	err := store.RunInTransaction(func(transactionStore types.ExpenseTransactionStore) error {
		if err := transactionStore.CreateExpense(expense); err != nil {
			return err
		}
		if err := transactionStore.CreateItem(item); err != nil {
			return err
		}
		return transactionStore.CreateItem(item)
	})
	require.Error(t, err)

	var expenseCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM expense WHERE id = $1", expenseID).Scan(&expenseCount))
	assert.Zero(t, expenseCount)

	var itemCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM item WHERE id = $1", itemID).Scan(&itemCount))
	assert.Zero(t, itemCount)
}

func TestRunInTransactionRestoresOutdatedBalanceAfterLaterFailure(t *testing.T) {
	db := openTestDB(t)
	store := expense.NewStore(db)
	groupID := uuid.New()
	existingBalance := &types.Balance{
		ID:             uuid.New(),
		SenderUserID:   uuid.New(),
		ReceiverUserID: uuid.New(),
		Share:          decimal.NewFromInt(10),
		GroupID:        groupID,
	}
	require.NoError(t, ensureBalanceParents(db, existingBalance))
	require.NoError(t, store.CreateBalances(groupID.String(), []*types.Balance{existingBalance}))
	defer deleteBalances(db, []*types.Balance{existingBalance})

	err := store.RunInTransaction(func(transactionStore types.ExpenseTransactionStore) error {
		if err := transactionStore.OutdateBalanceByGroupId(groupID.String()); err != nil {
			return err
		}

		duplicateBalance := &types.Balance{
			ID:             existingBalance.ID,
			SenderUserID:   uuid.New(),
			ReceiverUserID: uuid.New(),
			Share:          decimal.NewFromInt(5),
		}
		return transactionStore.CreateBalances(groupID.String(), []*types.Balance{duplicateBalance})
	})
	require.Error(t, err)

	var isOutdated bool
	require.NoError(t, db.QueryRow("SELECT is_outdated FROM balance WHERE id = $1", existingBalance.ID).Scan(&isOutdated))
	assert.False(t, isOutdated)
}

package store_test

import (
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		SplitRule:      "Equally",
	}
	item := types.Item{
		ID:        itemID,
		ExpenseID: expenseID,
		Name:      "duplicate item",
		Amount:    decimal.NewFromInt(1),
		Unit:      "each",
		UnitPrice: decimal.NewFromInt(10),
	}

	defer deleteExpense(db, expenseID)
	defer deleteItem(db, itemID)

	err := store.RunInTransaction(func(transactionStore types.ExpenseStore) error {
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
	require.NoError(t, store.CreateBalances(groupID.String(), []*types.Balance{existingBalance}))
	defer deleteBalances(db, []*types.Balance{existingBalance})

	err := store.RunInTransaction(func(transactionStore types.ExpenseStore) error {
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

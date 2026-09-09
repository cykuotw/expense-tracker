package expense

import (
	"testing"

	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestValidateExpenseMoneyChecksRoundedItemSubtotal(t *testing.T) {
	actorID := uuid.New()
	expense := types.Expense{
		PayByUserId: actorID,
		Currency:    "CAD",
		SubTotal:    decimal.RequireFromString("3.74"),
		Total:       decimal.RequireFromString("3.74"),
		SplitRule:   "Unequally",
	}
	items := []types.Item{{
		Amount:    decimal.RequireFromString("1.25"),
		UnitPrice: decimal.RequireFromString("2.99"),
	}}
	ledgers := []types.Ledger{{
		LenderUserID:   actorID,
		BorrowerUesrID: actorID,
		Share:          decimal.RequireFromString("3.74"),
	}}
	store := expenseStoreMock()

	require.NoError(t, validateExpenseMoney(store, expense, items, ledgers, actorID))

	expense.SubTotal = decimal.RequireFromString("3.73")
	require.ErrorIs(t, validateExpenseMoney(store, expense, items, ledgers, actorID), types.ErrInvalidMoney)
}

func TestValidateExpenseMoneyChecksViewerRelativeFullSplit(t *testing.T) {
	actorID := uuid.New()
	otherID := uuid.New()
	expense := types.Expense{
		PayByUserId: actorID,
		Currency:    "CAD",
		Total:       decimal.NewFromInt(10),
		SplitRule:   "You-Full",
	}
	ledgers := []types.Ledger{
		{LenderUserID: actorID, BorrowerUesrID: actorID, Share: decimal.NewFromInt(10)},
		{LenderUserID: actorID, BorrowerUesrID: otherID, Share: decimal.Zero},
	}
	store := expenseStoreMock()

	require.NoError(t, validateExpenseMoney(store, expense, nil, ledgers, actorID))

	expense.SplitRule = "Other-Full"
	require.ErrorIs(t, validateExpenseMoney(store, expense, nil, ledgers, actorID), types.ErrInvalidMoney)

	expense.SplitRule = "You-Full"
	ledgers[0].Share, ledgers[1].Share = decimal.Zero, decimal.NewFromInt(10)
	require.ErrorIs(t, validateExpenseMoney(store, expense, nil, ledgers, actorID), types.ErrInvalidMoney)
}

func TestValidateExpenseMoneySupportsThreeAmountDigits(t *testing.T) {
	actorID := uuid.New()
	store := expenseStoreMock()
	store.GetCurrencyAmountDigitsFn = func(currency string) (int32, error) {
		require.Equal(t, "KWD", currency)
		return 3, nil
	}
	expense := types.Expense{
		PayByUserId: actorID,
		Currency:    "KWD",
		Total:       decimal.RequireFromString("1.001"),
		SplitRule:   "Unequally",
	}
	ledgers := []types.Ledger{{
		LenderUserID:   actorID,
		BorrowerUesrID: actorID,
		Share:          decimal.RequireFromString("1.001"),
	}}

	require.NoError(t, validateExpenseMoney(store, expense, nil, ledgers, actorID))
}

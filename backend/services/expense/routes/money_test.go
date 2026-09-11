package expense

import (
	"testing"

	"expense-tracker/backend/types"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateExpenseMoneyUsesCurrencyPrecisionFromStore(t *testing.T) {
	store := expenseStoreMock()
	store.GetCurrencyAmountDigitsFn = func(currency string) (int32, error) {
		require.Equal(t, "KWD", currency)
		return 3, nil
	}
	expense := types.Expense{Currency: "KWD", Total: decimal.RequireFromString("1.001")}

	amountDigits, err := validateExpenseMoney(store, expense, nil)

	require.NoError(t, err)
	assert.Equal(t, int32(3), amountDigits)
}

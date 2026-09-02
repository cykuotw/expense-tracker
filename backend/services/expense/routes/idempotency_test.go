package expense

import (
	"crypto/sha256"
	"testing"

	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpenseCreateFingerprintPreservesLegacyOmittedDateShape(t *testing.T) {
	expense := types.Expense{
		Description: "Dinner", GroupID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		PayByUserId:   uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		ExpenseTypeID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		SubTotal:      decimal.NewFromInt(10), TaxFeeTip: decimal.NewFromInt(1), Total: decimal.NewFromInt(11),
		Currency: "CAD", SplitRule: "Equally", OccurredOn: "2026-08-31",
	}
	legacyJSON := `{"description":"Dinner","groupId":"00000000-0000-0000-0000-000000000001","payByUserId":"00000000-0000-0000-0000-000000000002","expenseTypeId":"00000000-0000-0000-0000-000000000003","providerName":"","subTotal":"10","taxFeeTip":"1","total":"11","currency":"CAD","invoiceUrl":"","splitRule":"Equally","items":[],"ledgers":[]}`
	expected := sha256.Sum256([]byte(legacyJSON))

	actual, err := expenseCreateFingerprint(expense, nil, nil, nil)

	require.NoError(t, err)
	assert.Equal(t, expected[:], actual)
}

func TestExpenseCreateFingerprintIncludesRequestedOccurredOn(t *testing.T) {
	expense := types.Expense{OccurredOn: "2026-08-31"}
	requested := expense.OccurredOn

	legacy, err := expenseCreateFingerprint(expense, nil, nil, nil)
	require.NoError(t, err)
	withDate, err := expenseCreateFingerprint(expense, nil, nil, &requested)
	require.NoError(t, err)

	assert.NotEqual(t, legacy, withDate)
}

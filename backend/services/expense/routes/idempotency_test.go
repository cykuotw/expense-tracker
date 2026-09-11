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

func TestExpenseCreateFingerprintPreservesOmittedDateShape(t *testing.T) {
	expense := types.Expense{
		Description: "Dinner", GroupID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		PayByUserId:   uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		ExpenseTypeID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		SubTotal:      decimal.NewFromInt(10), TaxFeeTip: decimal.NewFromInt(1), Total: decimal.NewFromInt(11),
		Currency: "CAD", AllocationMode: types.ExpenseAllocationEqual, OccurredOn: "2026-08-31",
	}
	canonicalJSON := `{"description":"Dinner","groupId":"00000000-0000-0000-0000-000000000001","payByUserId":"00000000-0000-0000-0000-000000000002","expenseTypeId":"00000000-0000-0000-0000-000000000003","providerName":"","subTotal":"10","taxFeeTip":"1","total":"11","currency":"CAD","invoiceUrl":"","allocationMode":"equal","items":[],"allocations":[]}`
	expected := sha256.Sum256([]byte(canonicalJSON))

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

func TestExpenseCreateFingerprintCanonicalizesAllocationOrderAndInputs(t *testing.T) {
	firstID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	secondID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	expense := types.Expense{AllocationMode: types.ExpenseAllocationPercentage}
	firstPercentage := int32(6_000)
	secondPercentage := int32(4_000)
	ordered := []types.ExpenseAllocation{
		{UserID: firstID, PercentageBasisPoints: &firstPercentage},
		{UserID: secondID, PercentageBasisPoints: &secondPercentage},
	}
	reversed := []types.ExpenseAllocation{ordered[1], ordered[0]}

	first, err := expenseCreateFingerprint(expense, nil, ordered, nil)
	require.NoError(t, err)
	second, err := expenseCreateFingerprint(expense, nil, reversed, nil)
	require.NoError(t, err)
	assert.Equal(t, first, second)

	changedPercentage := int32(5_000)
	changed, err := expenseCreateFingerprint(expense, nil, []types.ExpenseAllocation{
		{UserID: firstID, PercentageBasisPoints: &changedPercentage},
		{UserID: secondID, PercentageBasisPoints: &changedPercentage},
	}, nil)
	require.NoError(t, err)
	assert.NotEqual(t, first, changed)

	expense.AllocationMode = types.ExpenseAllocationEqual
	changedMode, err := expenseCreateFingerprint(expense, nil, []types.ExpenseAllocation{
		{UserID: firstID}, {UserID: secondID},
	}, nil)
	require.NoError(t, err)
	assert.NotEqual(t, first, changedMode)
}

func TestExpenseCreateFingerprintCanonicalizesItemsWithCollidingConcatenations(t *testing.T) {
	expense := types.Expense{}
	first := types.Item{Name: "a", Amount: decimal.RequireFromString("12"), Unit: "3", UnitPrice: decimal.RequireFromString("4")}
	second := types.Item{Name: "a1", Amount: decimal.RequireFromString("2"), Unit: "3", UnitPrice: decimal.RequireFromString("4")}

	ordered, err := expenseCreateFingerprint(expense, []types.Item{first, second}, nil, nil)
	require.NoError(t, err)
	reversed, err := expenseCreateFingerprint(expense, []types.Item{second, first}, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, ordered, reversed)
}

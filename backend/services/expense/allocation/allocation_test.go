package allocation_test

import (
	"testing"

	"expense-tracker/backend/services/expense/allocation"
	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMoneyChecksRoundedItemSubtotal(t *testing.T) {
	expense := types.Expense{SubTotal: decimal.RequireFromString("3.74"), Total: decimal.RequireFromString("3.74")}
	items := []types.Item{{Amount: decimal.RequireFromString("1.25"), UnitPrice: decimal.RequireFromString("2.99")}}

	require.NoError(t, allocation.ValidateMoney(expense, items, 2))
	expense.SubTotal = decimal.RequireFromString("3.73")
	require.ErrorIs(t, allocation.ValidateMoney(expense, items, 2), types.ErrInvalidMoney)
}

func TestParsePayloadRejectsInvalidParticipants(t *testing.T) {
	userID := uuid.NewString()
	tests := []types.ExpenseAllocationPayload{
		{Mode: types.ExpenseAllocationEqual},
		{Mode: types.ExpenseAllocationEqual, Participants: []types.ExpenseAllocationParticipantPayload{{UserID: userID}, {UserID: userID}}},
		{Mode: types.ExpenseAllocationEqual, Participants: []types.ExpenseAllocationParticipantPayload{{UserID: userID, Amount: decimalPointer("1")}}},
	}
	for _, payload := range tests {
		_, err := allocation.ParsePayload(uuid.New(), payload)
		require.Error(t, err)
	}
}

func TestDeriveLedgersEqualUsesStableParticipantOrder(t *testing.T) {
	expense := types.Expense{ID: uuid.New(), PayByUserId: uuid.New(), Total: decimal.NewFromInt(10), AllocationMode: types.ExpenseAllocationEqual}
	allocations := []types.ExpenseAllocation{
		{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
		{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000002")},
		{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000003")},
	}

	ledgers, err := allocation.DeriveLedgers(expense, allocations, 2)

	require.NoError(t, err)
	require.Len(t, ledgers, 3)
	assert.Equal(t, "3.34", ledgers[0].Share.StringFixed(2))
	assert.Equal(t, "3.33", ledgers[1].Share.StringFixed(2))
	assert.Equal(t, "3.33", ledgers[2].Share.StringFixed(2))
}

func TestDeriveLedgersPercentageUsesLargestRemainder(t *testing.T) {
	expense := types.Expense{ID: uuid.New(), PayByUserId: uuid.New(), Total: decimal.RequireFromString("0.05"), AllocationMode: types.ExpenseAllocationPercentage}
	allocations := []types.ExpenseAllocation{
		{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), PercentageBasisPoints: int32Pointer(3333)},
		{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), PercentageBasisPoints: int32Pointer(3333)},
		{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), PercentageBasisPoints: int32Pointer(3334)},
	}

	ledgers, err := allocation.DeriveLedgers(expense, allocations, 2)

	require.NoError(t, err)
	assert.Equal(t, "0.02", ledgers[0].Share.StringFixed(2))
	assert.Equal(t, "0.01", ledgers[1].Share.StringFixed(2))
	assert.Equal(t, "0.02", ledgers[2].Share.StringFixed(2))
}

func TestDeriveLedgersAddsAdjustmentsAfterEqualRemainder(t *testing.T) {
	expense := types.Expense{ID: uuid.New(), PayByUserId: uuid.New(), Total: decimal.RequireFromString("10.00"), AllocationMode: types.ExpenseAllocationAdjustment}
	allocations := []types.ExpenseAllocation{
		{UserID: uuid.New(), Amount: decimalPointer("1.00")},
		{UserID: uuid.New(), Amount: decimalPointer("0")},
		{UserID: uuid.New(), Amount: decimalPointer("0")},
	}

	ledgers, err := allocation.DeriveLedgers(expense, allocations, 2)

	require.NoError(t, err)
	assert.Equal(t, "4.00", ledgers[0].Share.StringFixed(2))
	assert.Equal(t, "3.00", ledgers[1].Share.StringFixed(2))
	assert.Equal(t, "3.00", ledgers[2].Share.StringFixed(2))
}

func TestDeriveLedgersRejectsModeSpecificInvalidInputs(t *testing.T) {
	userID := uuid.New()
	tests := []struct {
		mode        types.ExpenseAllocationMode
		allocations []types.ExpenseAllocation
	}{
		{types.ExpenseAllocationExact, []types.ExpenseAllocation{{UserID: userID, Amount: decimalPointer("9.99")}}},
		{types.ExpenseAllocationPercentage, []types.ExpenseAllocation{{UserID: userID, PercentageBasisPoints: int32Pointer(9999)}}},
		{types.ExpenseAllocationAdjustment, []types.ExpenseAllocation{{UserID: userID, Amount: decimalPointer("10.01")}}},
		{"unsupported", []types.ExpenseAllocation{{UserID: userID}}},
	}
	for _, test := range tests {
		expense := types.Expense{ID: uuid.New(), PayByUserId: uuid.New(), Total: decimal.RequireFromString("10.00"), AllocationMode: test.mode}
		_, err := allocation.DeriveLedgers(expense, test.allocations, 2)
		require.Error(t, err)
	}
}

func TestParticipantIDsDeduplicatesActorPayerAndParticipants(t *testing.T) {
	actorID := uuid.New()
	otherID := uuid.New()
	ids := allocation.ParticipantIDs(actorID, types.Expense{PayByUserId: actorID}, []types.ExpenseAllocation{{UserID: actorID}, {UserID: otherID}})

	assert.Equal(t, []uuid.UUID{actorID, otherID}, ids)
}

func decimalPointer(value string) *decimal.Decimal {
	amount := decimal.RequireFromString(value)
	return &amount
}

func int32Pointer(value int32) *int32 {
	return &value
}

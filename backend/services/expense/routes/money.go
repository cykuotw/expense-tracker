package expense

import (
	"expense-tracker/backend/services/expense/allocation"
	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

type currencyPrecisionStore interface {
	GetCurrencyAmountDigits(currency string) (int32, error)
}

func parseExpenseAllocationPayload(
	expenseID uuid.UUID,
	payload types.ExpenseAllocationPayload,
) ([]types.ExpenseAllocation, error) {
	return allocation.ParsePayload(expenseID, payload)
}

func validateExpenseMoney(
	store currencyPrecisionStore,
	expense types.Expense,
	items []types.Item,
) (int32, error) {
	amountDigits, err := store.GetCurrencyAmountDigits(expense.Currency)
	if err != nil {
		return 0, err
	}
	if err := allocation.ValidateMoney(expense, items, amountDigits); err != nil {
		return 0, err
	}
	return amountDigits, nil
}

func deriveExpenseLedgers(
	expense types.Expense,
	allocations []types.ExpenseAllocation,
	amountDigits int32,
) ([]types.Ledger, error) {
	return allocation.DeriveLedgers(expense, allocations, amountDigits)
}

func expenseParticipantIDs(
	actorID uuid.UUID,
	expense types.Expense,
	allocations []types.ExpenseAllocation,
) []uuid.UUID {
	return allocation.ParticipantIDs(actorID, expense, allocations)
}

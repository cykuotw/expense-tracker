package expense

import (
	"expense-tracker/backend/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"unicode/utf8"
)

func validateCategoryBreakdown(expense types.Expense, rows []types.ExpenseCategoryAmount, digits int32) error {
	if len(rows) == 0 {
		return nil
	}
	if len(rows) < 2 || len(rows) > 20 || rows[0].ExpenseTypeID != expense.ExpenseTypeID {
		return types.ErrInvalidCategoryBreakdown
	}
	seen := make(map[uuid.UUID]bool, len(rows))
	total := decimal.Zero
	for _, row := range rows {
		if row.ExpenseTypeID == uuid.Nil || seen[row.ExpenseTypeID] || !row.Amount.IsPositive() ||
			!row.Amount.Equal(row.Amount.Round(digits)) || utf8.RuneCountInString(row.Note) > 200 {
			return types.ErrInvalidCategoryBreakdown
		}
		seen[row.ExpenseTypeID] = true
		total = total.Add(row.Amount)
	}
	if !total.Equal(expense.Total) {
		return types.ErrInvalidCategoryBreakdown
	}
	return nil
}

func effectiveCategoryBreakdown(expense *types.Expense, rows []types.ExpenseCategoryAmount, expenseType *types.ExpenseType) []types.ExpenseCategoryAmount {
	if len(rows) > 0 {
		return rows
	}
	row := types.ExpenseCategoryAmount{ExpenseTypeID: expense.ExpenseTypeID, Amount: expense.Total}
	if expenseType != nil {
		row.ExpenseType = expenseType.Name
		row.Category = expenseType.Category
	}
	return []types.ExpenseCategoryAmount{row}
}

package types

import (
	"errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrInvalidCategoryBreakdown = errors.New("category amounts must use 2–20 different valid categories, have positive amounts at the currency precision, and add up to the total; notes may contain up to 200 characters")
var ErrCategoryBreakdownRequired = errors.New("this expense has multiple categories; reload it before editing to preserve its category amounts")

type ExpenseCategoryAmount struct {
	ExpenseTypeID uuid.UUID       `json:"expenseTypeId"`
	Amount        decimal.Decimal `json:"amount"`
	Note          string          `json:"note,omitempty"`
	ExpenseType   string          `json:"expenseType,omitempty"`
	Category      string          `json:"category,omitempty"`
}

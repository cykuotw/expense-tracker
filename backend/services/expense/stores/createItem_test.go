package store_test

import (
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCreateItem(t *testing.T) {
	// prepare test data
	db := openTestDB(t)
	store := expense.NewStore(db)

	// define test cases
	type testcase struct {
		name        string
		mockItem    types.Item
		expectFail  bool
		expectError error
	}

	subtests := []testcase{
		{
			name: "valid",
			mockItem: types.Item{
				ID: uuid.New(), ExpenseID: uuid.New(), Description: "test name",
				Quantity: decimalPointer("3.7"), Unit: stringPointer("ea"),
				UnitPrice: decimalPointer("2.9"), LineTotal: decimal.RequireFromString("10.73"),
			},
			expectFail:  false,
			expectError: nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			parent := newTestExpense(test.mockItem.ExpenseID)
			assert.NoError(t, insertExpense(db, parent))
			defer deleteExpense(db, parent.ID)
			err := store.CreateItem(test.mockItem)
			defer deleteItem(db, test.mockItem.ID)

			assert.Equal(t, test.expectError, err)
		})
	}
}

func decimalPointer(value string) *decimal.Decimal {
	parsed := decimal.RequireFromString(value)
	return &parsed
}

func stringPointer(value string) *string {
	return &value
}

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
				ID:        uuid.New(),
				ExpenseID: uuid.New(),
				Name:      "test name",
				Amount:    decimal.NewFromFloat(3.7),
				Unit:      "ea",
				UnitPrice: decimal.NewFromFloat(2.9),
			},
			expectFail:  false,
			expectError: nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			err := store.CreateItem(test.mockItem)
			defer deleteItem(db, test.mockItem.ID)

			assert.Equal(t, test.expectError, err)
		})
	}
}

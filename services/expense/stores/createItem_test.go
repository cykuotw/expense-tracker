package store_test

import (
	"expense-tracker/config"
	"expense-tracker/db"
	expense "expense-tracker/services/expense/stores"
	"expense-tracker/types"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCreateItem(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
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

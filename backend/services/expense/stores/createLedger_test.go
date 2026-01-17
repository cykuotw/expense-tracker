package store_test

import (
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCreateLedger(t *testing.T) {
	// prepare test data
	db := openTestDB(t)
	store := expense.NewStore(db)

	// define test cases
	type testcase struct {
		name        string
		mockLedger  types.Ledger
		expectFail  bool
		expectError error
	}

	subtests := []testcase{
		{
			name: "valid",
			mockLedger: types.Ledger{
				ID:             uuid.New(),
				ExpenseID:      uuid.New(),
				LenderUserID:   uuid.New(),
				BorrowerUesrID: uuid.New(),
				Share:          decimal.NewFromFloat(5.597),
			},
			expectFail:  false,
			expectError: nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			err := store.CreateLedger(test.mockLedger)
			defer deleteLedger(db, test.mockLedger.ID)

			assert.Equal(t, test.expectError, err)
		})
	}
}

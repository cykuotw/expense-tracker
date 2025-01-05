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

func TestCreateLedger(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
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

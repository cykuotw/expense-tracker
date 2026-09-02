package store_test

import (
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCreateExpense(t *testing.T) {
	// prepare test data
	db := openTestDB(t)
	store := expense.NewStore(db)

	// define test cases
	type testcase struct {
		name        string
		mockExpense types.Expense
		expectFail  bool
		expectError error
	}

	subtests := []testcase{
		{
			name: "valid",
			mockExpense: types.Expense{
				ID:             uuid.New(),
				Description:    "test desc",
				GroupID:        mockGroupID,
				CreateByUserID: mockCreatorID,
				PayByUserId:    mockPayerID,
				ExpenseTypeID:  uuid.New(),
				CreateTime:     time.Now(),
				ProviderName:   "test prov",
				IsSettled:      false,
				SubTotal:       decimal.NewFromFloat(20.01),
				TaxFeeTip:      decimal.NewFromFloat(1.01),
				Total:          decimal.NewFromFloat(21.02),
				Currency:       "CAD",
				InvoicePicUrl:  "http://mockpic.url.com",
				SplitRule:      "Unequally",
				OccurredOn:     "2026-08-31",
			},
			expectFail:  false,
			expectError: nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			assert.NoError(t, ensureExpenseParents(db, test.mockExpense))
			startedAt := time.Now().UTC().Add(-time.Second)
			err := store.CreateExpense(test.mockExpense)
			finishedAt := time.Now().UTC().Add(time.Second)
			defer deleteExpense(db, test.mockExpense.ID)

			assert.Equal(t, test.expectError, err)
			created := selectExpenseByID(db, test.mockExpense.ID)
			for _, instant := range []time.Time{
				created.CreateTime,
				created.UpdateTime,
				created.ExpenseTime,
			} {
				assert.False(t, instant.IsZero())
				assert.False(t, instant.Before(startedAt))
				assert.False(t, instant.After(finishedAt))
			}
			assert.Equal(t, test.mockExpense.OccurredOn, created.OccurredOn)
		})
	}
}

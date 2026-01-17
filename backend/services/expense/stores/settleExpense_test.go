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

func TestSettleExpenseByGroupId(t *testing.T) {
	db := openTestDB(t)
	store := expense.NewStore(db)

	type testcase struct {
		name        string
		mockExpense []types.Expense
		mockGroupId string
		expectFail  bool
		expectError error
	}

	subtests := []testcase{
		{
			name: "valid",
			mockExpense: []types.Expense{
				{
					ID:             uuid.New(),
					Description:    "test desc",
					GroupID:        mockGroupID,
					CreateByUserID: mockCreatorID,
					PayByUserId:    mockPayerID,
					ExpenseTypeID:  uuid.New(),
					CreateTime:     time.Now(),
					IsSettled:      false,
					IsDeleted:      false,
					Total:          decimal.NewFromFloat(21.02),
					Currency:       "CAD",
					SplitRule:      "Unequally",
				},
			},
			mockGroupId: mockGroupID.String(),
			expectFail:  false,
			expectError: nil,
		},
		{
			name: "invalid-unmatched group id",
			mockExpense: []types.Expense{
				{
					ID:             uuid.New(),
					Description:    "test desc",
					GroupID:        mockGroupID,
					CreateByUserID: mockCreatorID,
					PayByUserId:    mockPayerID,
					ExpenseTypeID:  uuid.New(),
					CreateTime:     time.Now(),
					IsSettled:      false,
					IsDeleted:      false,
					Total:          decimal.NewFromFloat(21.02),
					Currency:       "CAD",
					SplitRule:      "Unequally",
				},
			},
			mockGroupId: uuid.NewString(),
			expectFail:  true,
			expectError: nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			for _, exp := range test.mockExpense {
				insertExpense(db, exp)
				defer deleteExpense(db, exp.ID)
			}

			err := store.SettleExpenseByGroupId(test.mockGroupId)

			expenses := selectExpense(db, uuid.MustParse(test.mockGroupId))
			if test.expectFail {
				assert.ErrorIs(t, err, test.expectError)
			} else {
				assert.Nil(t, err)
				for _, exp := range expenses {
					assert.True(t, exp.IsSettled)
					assert.LessOrEqual(t,
						time.Until(exp.SettleTime).Seconds(),
						time.Duration(1*time.Second).Seconds())
				}
			}
		})
	}
}

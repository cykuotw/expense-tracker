package store_test

import (
	"expense-tracker/config"
	"expense-tracker/db"
	expense "expense-tracker/services/expense/stores"
	"expense-tracker/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestDeleteExpense(t *testing.T) {
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	store := expense.NewStore(db)

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
				IsSettled:      false,
				IsDeleted:      false,
				Total:          decimal.NewFromFloat(21.02),
				Currency:       "CAD",
				SplitRule:      "Unequally",
			},
			expectFail:  false,
			expectError: nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			insertExpense(db, test.mockExpense)
			defer deleteExpense(db, test.mockExpense.ID)

			err := store.DeleteExpense(test.mockExpense)

			deletedExpense := selectExpense(db, test.mockExpense.GroupID)[0]

			if !test.expectFail {
				assert.Nil(t, err)
				assert.True(t, deletedExpense.IsDeleted)
				assert.LessOrEqual(t,
					deletedExpense.DeleteTime.Sub(time.Now()).Seconds(),
					time.Duration(1*time.Second).Seconds())
			}
		})
	}
}

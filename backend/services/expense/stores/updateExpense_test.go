package store_test

import (
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestUpdateExpenseSettleInGroup(t *testing.T) {
	// prepare test data
	db := openTestDB(t)
	store := expense.NewStore(db)

	toBeSettleExpCount := 5
	toBeSettleGroupID := uuid.New()
	toBeSettleExpenseIDs := []uuid.UUID{}

	unsettledExpCount := 3
	unsettledGroupID := uuid.New()
	unsettledExpenseIDs := []uuid.UUID{}

	for i := 0; i < toBeSettleExpCount; i++ {
		id := uuid.New()
		toBeSettleExpenseIDs = append(toBeSettleExpenseIDs, id)

		expense := types.Expense{
			ID:             id,
			Description:    "to be settle " + strconv.Itoa(i),
			GroupID:        toBeSettleGroupID,
			CreateByUserID: mockCreatorID,
			PayByUserId:    mockPayerID,
			CreateTime:     time.Now(),
			ExpenseTypeID:  mockExpenseTypeID,
			IsSettled:      false,
			Total:          decimal.NewFromFloat(99.37 + 0.37*float64(i)),
			Currency:       "CAD",
			SplitRule:      "Equally",
		}
		insertExpense(db, expense)
	}
	defer deleteExpenses(db, toBeSettleExpenseIDs)

	for i := 0; i < unsettledExpCount; i++ {
		id := uuid.New()
		unsettledExpenseIDs = append(unsettledExpenseIDs, id)

		expense := types.Expense{
			ID:             id,
			Description:    "unsettle " + strconv.Itoa(i),
			GroupID:        unsettledGroupID,
			CreateByUserID: mockCreatorID,
			PayByUserId:    mockPayerID,
			CreateTime:     time.Now(),
			ExpenseTypeID:  mockExpenseTypeID,
			IsSettled:      false,
			Total:          decimal.NewFromFloat(99.37 + 0.37*float64(i)),
			Currency:       "CAD",
			SplitRule:      "Equally",
		}
		insertExpense(db, expense)
	}
	defer deleteExpenses(db, unsettledExpenseIDs)

	// prepare test case
	type testcase struct {
		name                  string
		groupID               string
		expectFail            bool
		expectSettledLength   int
		expectUnsettledLength int
		expectSettledIDs      []uuid.UUID
		expectUnsettledIDs    []uuid.UUID
	}

	subtests := []testcase{
		{
			name:                  "valid",
			groupID:               toBeSettleGroupID.String(),
			expectFail:            false,
			expectSettledLength:   toBeSettleExpCount,
			expectUnsettledLength: unsettledExpCount,
			expectSettledIDs:      toBeSettleExpenseIDs,
			expectUnsettledIDs:    unsettledExpenseIDs,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			err := store.UpdateExpenseSettleInGroup(test.groupID)

			assert.Nil(t, err)

			expenseSettled := selectExpense(db, toBeSettleGroupID)
			expenseUnsettled := selectExpense(db, unsettledGroupID)

			assert.Equal(t, test.expectSettledLength, len(expenseSettled))
			for _, exp := range expenseSettled {
				assert.Contains(t, test.expectSettledIDs, exp.ID)
			}

			assert.Equal(t, test.expectUnsettledLength, len(expenseUnsettled))
			for _, exp := range expenseUnsettled {
				assert.Contains(t, test.expectUnsettledIDs, exp.ID)
			}

		})
	}
}

func TestUpdateExpense(t *testing.T) {
	// prepare test data
	db := openTestDB(t)
	store := expense.NewStore(db)

	mockExpense := types.Expense{
		ID:             uuid.New(),
		Description:    "original desc",
		GroupID:        mockGroupID,
		CreateByUserID: mockCreatorID,
		PayByUserId:    mockPayerID,
		UpdateTime:     time.Now(),
		ExpenseTime:    time.Now(),
		ExpenseTypeID:  mockExpenseTypeID,
		IsSettled:      false,
		Total:          decimal.NewFromFloat(99.37 + 0.37*8.3),
		Currency:       "CAD",
		SplitRule:      "Equally",
	}
	insertExpense(db, mockExpense)
	defer deleteExpense(db, mockExpense.ID)

	mockExpenseModified := mockExpense
	mockExpenseModified.Description = "new desc"

	// prepare test case
	type testcase struct {
		name          string
		expense       types.Expense
		expectFail    bool
		expectExpense types.Expense
		expectError   error
	}

	subtests := []testcase{
		{
			name:          "valid",
			expense:       mockExpenseModified,
			expectFail:    false,
			expectExpense: mockExpenseModified,
			expectError:   nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			err := store.UpdateExpense(test.expense)

			assert.Nil(t, err)

			expense := selectExpenseByID(db, test.expense.ID)
			assert.Equal(t, test.expectExpense.Description, expense.Description)
		})
	}
}

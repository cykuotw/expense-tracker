package store_test

import (
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"fmt"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGetItemsByExpenseID(t *testing.T) {
	// prepare test data
	db := openTestDB(t)
	store := expense.NewStore(db)

	mockExpenseID := uuid.New()

	testSetSize := 13
	itemIDs := []uuid.UUID{}
	for i := 0; i < testSetSize; i++ {
		id := uuid.New()
		itemIDs = append(itemIDs, id)

		item := types.Item{
			ID:        id,
			ExpenseID: mockExpenseID,
			Name:      "test " + strconv.Itoa(i),
			Amount:    decimal.NewFromFloat(3.66 + float64(i)),
			Unit:      "lbs",
			UnitPrice: decimal.NewFromFloat(0.7 + float64(i)),
		}
		insertItem(db, item)
	}
	defer deleteItems(db, itemIDs)

	// prepare test case
	type testcase struct {
		name         string
		expenseID    string
		expectFail   bool
		expectLength int
		expectItemID []uuid.UUID
		expectError  error
	}

	subtests := []testcase{
		{
			name:         "valid",
			expenseID:    mockExpenseID.String(),
			expectFail:   false,
			expectLength: testSetSize,
			expectItemID: itemIDs,
			expectError:  nil,
		},
		{
			name:         "non-existing expense id",
			expenseID:    uuid.NewString(),
			expectFail:   false,
			expectLength: 0,
			expectItemID: []uuid.UUID{},
			expectError:  nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			itemList, err := store.GetItemsByExpenseID(test.expenseID)
			fmt.Println(itemList)

			if test.expectFail {
				assert.Nil(t, itemList)
				assert.Equal(t, test.expectError, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, itemList)
				assert.Len(t, itemList, test.expectLength)
				for _, item := range itemList {
					assert.Contains(t, test.expectItemID, item.ID)
				}
			}
		})
	}
}

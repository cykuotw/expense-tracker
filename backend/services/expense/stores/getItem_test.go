package store_test

import (
	"expense-tracker/backend/config"
	"expense-tracker/backend/db"
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGetItemsByExpenseID(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
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
			name:         "invalid expense id",
			expenseID:    uuid.NewString(),
			expectFail:   true,
			expectLength: 0,
			expectItemID: nil,
			expectError:  types.ErrExpenseNotExist,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			itemList, err := store.GetItemsByExpenseID(test.expenseID)

			if test.expectFail {
				assert.Nil(t, itemList)
				assert.Equal(t, test.expectError, err)
			} else {
				assert.NotNil(t, itemList)
				assert.Equal(t, test.expectLength, len(itemList))
				for i := 0; i < test.expectLength; i++ {
					assert.Contains(t, test.expectItemID, itemList[i].ID)
				}
			}
		})
	}
}

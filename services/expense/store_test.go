package expense_test

import (
	"database/sql"
	"expense-tracker/config"
	"expense-tracker/db"
	"expense-tracker/services/expense"
	"expense-tracker/types"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCreateExpense(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
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
			},
			expectFail:  false,
			expectError: nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			err := store.CreateExpense(test.mockExpense)
			defer deleteExpense(db, test.mockExpense.ID)

			assert.Equal(t, test.expectError, err)
		})
	}
}

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

var mockGroupID = uuid.New()
var mockCreatorID = uuid.New()
var mockPayerID = uuid.New()

func deleteExpense(db *sql.DB, expenseId uuid.UUID) {
	query := fmt.Sprintf("DELETE FROM expense WHERE id='%s';", expenseId)
	db.Exec(query)
}

func deleteItem(db *sql.DB, itemID uuid.UUID) {
	query := fmt.Sprintf("DELETE FROM item WHERE id='%s';", itemID)
	db.Exec(query)
}

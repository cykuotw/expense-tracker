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

func TestGetExpenseByID(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	store := expense.NewStore(db)
	mockExpenseID := uuid.New()
	mockExpense := types.Expense{
		ID:             mockExpenseID,
		Description:    "test desc",
		GroupID:        mockGroupID,
		CreateByUserID: mockCreatorID,
		PayByUserId:    mockPayerID,
		ExpenseTypeID:  uuid.New(),
		CreateTime:     time.Now(),
		ProviderName:   "test providder",
		IsSettled:      false,
		SubTotal:       decimal.NewFromFloat(10.28),
		TaxFeeTip:      decimal.NewFromFloat(1.49),
		Total:          decimal.NewFromFloat(11.77),
		Currency:       "CAD",
		InvoicePicUrl:  "https://test.com",
	}
	insertExpense(db, mockExpense)
	defer deleteExpense(db, mockExpenseID)

	// define test cases
	type testcase struct {
		name          string
		mockExpenseID string
		expectFail    bool
		expectError   error
	}

	subtests := []testcase{
		{
			name:          "valid",
			mockExpenseID: mockExpenseID.String(),
			expectFail:    false,
			expectError:   nil,
		},
		{
			name:          "invalid id",
			mockExpenseID: uuid.NewString(),
			expectFail:    true,
			expectError:   types.ErrExpenseNotExist,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			expense, err := store.GetExpenseByID(test.mockExpenseID)

			if test.expectFail {
				assert.Nil(t, expense)
				assert.Equal(t, test.expectError, err)
			} else {
				assert.NotNil(t, expense)
				assert.Equal(t, test.mockExpenseID, expense.ID.String())
				assert.Nil(t, err)
			}
		})
	}
}

var mockGroupID = uuid.New()
var mockCreatorID = uuid.New()
var mockPayerID = uuid.New()

func insertExpense(db *sql.DB, expense types.Expense) {
	createTime := expense.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	query := fmt.Sprintf(
		"INSERT INTO expense ("+
			"id, description, group_id, "+
			"create_by_user_id, pay_by_user_id, provider_name, "+
			"exp_type_id, is_settled, "+
			"sub_total, tax_fee_tip, total, "+
			"currency, invoice_pic_url, create_time_utc"+
			") VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%t', "+
			"'%s', '%s', '%s', '%s', '%s', '%s')",
		expense.ID, expense.Description, expense.GroupID,
		expense.CreateByUserID, expense.PayByUserId, expense.ProviderName,
		expense.ExpenseTypeID, false,
		expense.SubTotal.String(), expense.TaxFeeTip.String(), expense.Total.String(),
		expense.Currency, expense.InvoicePicUrl, createTime,
	)

	db.Exec(query)
}

func deleteExpense(db *sql.DB, expenseId uuid.UUID) {
	query := fmt.Sprintf("DELETE FROM expense WHERE id='%s';", expenseId)
	db.Exec(query)
}

func deleteItem(db *sql.DB, itemID uuid.UUID) {
	query := fmt.Sprintf("DELETE FROM item WHERE id='%s';", itemID)
	db.Exec(query)
}

func deleteLedger(db *sql.DB, ledgerID uuid.UUID) {
	query := fmt.Sprintf("DELETE FROM ledger WHERE id='%s';", ledgerID)
	db.Exec(query)
}

package expense_test

import (
	"database/sql"
	"expense-tracker/config"
	"expense-tracker/db"
	"expense-tracker/services/expense"
	"expense-tracker/types"
	"fmt"
	"strconv"
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
				SplitRule:      "Unequally",
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

func TestGetExpenseList(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	store := expense.NewStore(db)

	testSetSize := 60

	now := time.Now()
	interval := 10 * time.Minute
	idList := []uuid.UUID{}
	for i := 0; i < testSetSize; i++ {
		duration := time.Duration(i) * interval
		t := now.Add(duration)

		id := uuid.New()
		idList = append(idList, id)

		exp := types.Expense{
			ID:             id,
			Description:    "test desc " + strconv.Itoa(i),
			GroupID:        mockGroupID,
			CreateByUserID: mockCreatorID,
			PayByUserId:    mockPayerID,
			ExpenseTypeID:  mockExpenseTypeID,
			CreateTime:     t,
			IsSettled:      false,
			Total:          decimal.NewFromFloat(10.112),
			Currency:       "CAD",
			SplitRule:      "Equally",
		}

		insertExpense(db, exp)
	}
	defer deleteExpenses(db, idList)

	// prepare test case
	type testcase struct {
		name               string
		groupID            string
		totalPage          int64
		expectFail         bool
		expectExpenseCount []int
		expectExpenseID    [][]uuid.UUID
		expectError        []error
	}

	subtests := []testcase{
		{
			name:               "valid",
			groupID:            mockGroupID.String(),
			totalPage:          4,
			expectFail:         false,
			expectExpenseCount: []int{25, 25, 10, 0},
			expectExpenseID: [][]uuid.UUID{
				idList[:25],
				idList[25:50],
				idList[50:60],
				nil,
			},
			expectError: []error{nil, nil, nil, types.ErrNoRemainingExpenses},
		},
		{
			name:               "invalid group id",
			groupID:            uuid.NewString(),
			totalPage:          1,
			expectFail:         true,
			expectExpenseCount: nil,
			expectExpenseID:    nil,
			expectError:        []error{types.ErrNoRemainingExpenses},
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			var page int64
			for page = 0; page < test.totalPage; page++ {
				expenseList, err := store.GetExpenseList(test.groupID, page)

				if test.expectFail {
					assert.Nil(t, expenseList)
					assert.Equal(t, test.expectError[0], err)
				} else {
					if err == nil {
						assert.Equal(t, test.expectExpenseCount[page], len(expenseList))
					} else {
						assert.Equal(t, test.expectError[page], err)
					}

					for i, exp := range expenseList {
						assert.Equal(t, test.expectExpenseID[page][i], exp.ID)
					}
				}
			}
		})
	}
}

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

func TestGetLedgersByExpenseID(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	store := expense.NewStore(db)

	mockExpenseID := uuid.New()

	testSetSize := 13
	ledgerIDs := []uuid.UUID{}
	for i := 0; i < testSetSize; i++ {
		id := uuid.New()
		ledger := types.Ledger{
			ID:             id,
			ExpenseID:      mockExpenseID,
			LenderUserID:   uuid.New(),
			BorrowerUesrID: uuid.New(),
			Share:          decimal.NewFromFloat(5.33 + float64(i)),
		}
		insertLedger(db, ledger)
		ledgerIDs = append(ledgerIDs, id)
	}
	defer deleteLedgers(db, ledgerIDs)

	// prepare test case
	type testcase struct {
		name           string
		expenseID      string
		expectFail     bool
		expectLength   int
		expectLedgerID []uuid.UUID
		expectError    error
	}

	subtests := []testcase{
		{
			name:           "valid",
			expenseID:      mockExpenseID.String(),
			expectFail:     false,
			expectLength:   testSetSize,
			expectLedgerID: ledgerIDs,
			expectError:    nil,
		},
		{
			name:           "invalid expense id",
			expenseID:      uuid.NewString(),
			expectFail:     true,
			expectLength:   0,
			expectLedgerID: nil,
			expectError:    types.ErrExpenseNotExist,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			ledgerList, err := store.GetLedgersByExpenseID(test.expenseID)

			if test.expectFail {
				assert.Nil(t, ledgerList)
				assert.Equal(t, test.expectError, err)
			} else {
				assert.NotNil(t, ledgerList)
				assert.Equal(t, test.expectLength, len(ledgerList))
				for i := 0; i < test.expectLength; i++ {
					assert.Contains(t, test.expectLedgerID, ledgerList[i].ID)
				}
			}
		})
	}
}

func TestGetLedgerUnsettledFromGroup(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	store := expense.NewStore(db)

	unsettledExpenseCount := 3
	settledExpenseCount := 5

	unsettledExpenseIDs := []uuid.UUID{}
	settledExpenseIDs := []uuid.UUID{}

	unsettledLedgerCount := 2
	settledLedgerCount := 5

	unsettledLedgerIDs := []uuid.UUID{}
	settledLedgerIDs := []uuid.UUID{}

	for i := 0; i < unsettledExpenseCount; i++ {
		// unsettled
		expID := uuid.New()
		unsettledExpenseIDs = append(unsettledExpenseIDs, expID)
		expense := types.Expense{
			ID:             expID,
			Description:    "unsettled test " + strconv.Itoa(i),
			GroupID:        mockGroupID,
			CreateByUserID: mockCreatorID,
			CreateTime:     time.Now(),
			ExpenseTypeID:  mockExpenseTypeID,
			IsSettled:      false,
			Total:          decimal.NewFromFloat(99.37 + 0.37*float64(i)),
			Currency:       "CAD",
			SplitRule:      "Equally",
		}
		insertExpense(db, expense)

		for j := 0; j < unsettledLedgerCount; j++ {
			ledgerID := uuid.New()
			unsettledLedgerIDs = append(unsettledLedgerIDs, ledgerID)
			ledger := types.Ledger{
				ID:             ledgerID,
				ExpenseID:      expID,
				LenderUserID:   uuid.New(),
				BorrowerUesrID: uuid.New(),
				Share:          decimal.NewFromFloat(77.61 + 0.19*float64(i+j)),
			}
			insertLedger(db, ledger)
		}
	}
	defer deleteExpenses(db, unsettledExpenseIDs)
	defer deleteLedgers(db, unsettledLedgerIDs)

	for i := 0; i < settledExpenseCount; i++ {
		// settled
		expID := uuid.New()
		settledExpenseIDs = append(settledExpenseIDs, expID)
		expense := types.Expense{
			ID:             expID,
			Description:    "settled test " + strconv.Itoa(i),
			GroupID:        mockGroupID,
			CreateByUserID: mockCreatorID,
			CreateTime:     time.Now(),
			ExpenseTypeID:  mockExpenseTypeID,
			IsSettled:      true,
			Total:          decimal.NewFromFloat(99.37 + 0.37*float64(i)),
			Currency:       "CAD",
		}
		insertExpense(db, expense)

		for j := 0; j < settledLedgerCount; j++ {
			ledgerID := uuid.New()
			settledLedgerIDs = append(settledLedgerIDs, ledgerID)
			ledger := types.Ledger{
				ID:             ledgerID,
				ExpenseID:      expID,
				LenderUserID:   uuid.New(),
				BorrowerUesrID: uuid.New(),
				Share:          decimal.NewFromFloat(77.61 + 0.19*float64(i+j)),
			}
			insertLedger(db, ledger)
		}
	}
	defer deleteExpenses(db, settledExpenseIDs)
	defer deleteLedgers(db, settledLedgerIDs)

	// prepare test case
	type testcase struct {
		name           string
		groupID        string
		expectFail     bool
		expectLength   int
		expectLedgerID []uuid.UUID
		expectError    error
	}

	subtests := []testcase{
		{
			name:           "valid",
			groupID:        mockGroupID.String(),
			expectFail:     false,
			expectLength:   unsettledExpenseCount * unsettledLedgerCount,
			expectLedgerID: unsettledLedgerIDs,
			expectError:    nil,
		},
		{
			name:           "invalid group id",
			groupID:        uuid.NewString(),
			expectFail:     true,
			expectLength:   0,
			expectLedgerID: nil,
			expectError:    nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			ledgerList, err := store.GetLedgerUnsettledFromGroup(test.groupID)

			if test.expectFail {
				assert.NotNil(t, ledgerList)
				assert.Empty(t, ledgerList)
				assert.Equal(t, test.expectError, err)
			} else {
				assert.NotNil(t, ledgerList)
				assert.NotEmpty(t, ledgerList)
				assert.Nil(t, err)
				for _, ledger := range ledgerList {
					assert.Contains(t, test.expectLedgerID, ledger.ID)
				}
			}
		})
	}
}

func TestUpdateExpenseSettleInGroup(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
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
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
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

var mockGroupID = uuid.New()
var mockCreatorID = uuid.New()
var mockPayerID = uuid.New()
var mockExpenseTypeID = uuid.New()

func selectExpense(db *sql.DB, groupID uuid.UUID) []*types.Expense {
	query := fmt.Sprintf(
		"SELECT * FROM expense "+
			"WHERE group_id = '%s' "+
			"ORDER BY create_time_utc ASC;",
		groupID,
	)
	rows, _ := db.Query(query)
	defer rows.Close()

	expList := []*types.Expense{}

	for rows.Next() {
		expense := new(types.Expense)
		rows.Scan(
			&expense.ID,
			&expense.Description,
			&expense.GroupID,
			&expense.CreateByUserID,
			&expense.PayByUserId,
			&expense.ProviderName,
			&expense.ExpenseTypeID,
			&expense.IsSettled,
			&expense.SubTotal,
			&expense.TaxFeeTip,
			&expense.Total,
			&expense.Currency,
			&expense.InvoicePicUrl,
			&expense.CreateTime,
			&expense.UpdateTime,
			&expense.ExpenseTime,
			&expense.SplitRule,
		)
		expList = append(expList, expense)
	}

	return expList
}

func selectExpenseByID(db *sql.DB, expenseID uuid.UUID) *types.Expense {
	query := fmt.Sprintf(
		"SELECT * FROM expense "+
			"WHERE id = '%s';",
		expenseID,
	)
	rows, _ := db.Query(query)
	defer rows.Close()

	expense := new(types.Expense)

	for rows.Next() {
		rows.Scan(
			&expense.ID,
			&expense.Description,
			&expense.GroupID,
			&expense.CreateByUserID,
			&expense.PayByUserId,
			&expense.ProviderName,
			&expense.ExpenseTypeID,
			&expense.IsSettled,
			&expense.SubTotal,
			&expense.TaxFeeTip,
			&expense.Total,
			&expense.Currency,
			&expense.InvoicePicUrl,
			&expense.CreateTime,
			&expense.UpdateTime,
			&expense.ExpenseTime,
			&expense.SplitRule,
		)
	}

	return expense
}

func insertExpense(db *sql.DB, expense types.Expense) {
	createTime := expense.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	query := fmt.Sprintf(
		"INSERT INTO expense ("+
			"id, description, group_id, "+
			"create_by_user_id, pay_by_user_id, provider_name, "+
			"exp_type_id, is_settled, "+
			"sub_total, tax_fee_tip, total, "+
			"currency, invoice_pic_url, create_time_utc, split_rule"+
			") VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%t', "+
			"'%s', '%s', '%s', '%s', '%s', '%s', '%s')",
		expense.ID, expense.Description, expense.GroupID,
		expense.CreateByUserID, expense.PayByUserId, expense.ProviderName,
		expense.ExpenseTypeID, expense.IsSettled,
		expense.SubTotal.String(), expense.TaxFeeTip.String(), expense.Total.String(),
		expense.Currency, expense.InvoicePicUrl, createTime, expense.SplitRule,
	)

	db.Exec(query)
}

func deleteExpense(db *sql.DB, expenseId uuid.UUID) {
	query := fmt.Sprintf("DELETE FROM expense WHERE id='%s';", expenseId)
	db.Exec(query)
}

func deleteExpenses(db *sql.DB, expenseIds []uuid.UUID) {
	for _, id := range expenseIds {
		deleteExpense(db, id)
	}
}

func insertItem(db *sql.DB, item types.Item) {
	query := fmt.Sprintf(
		"INSERT INTO item ("+
			"id, expense_id, name, amount, unit, unit_price"+
			") VALUES ('%s', '%s', '%s', '%s', '%s', '%s')",
		item.ID, item.ExpenseID, item.Name, item.Amount, item.Unit, item.UnitPrice,
	)
	db.Exec(query)
}

func deleteItem(db *sql.DB, itemID uuid.UUID) {
	query := fmt.Sprintf("DELETE FROM item WHERE id='%s';", itemID)
	db.Exec(query)
}

func deleteItems(db *sql.DB, itemIDs []uuid.UUID) {
	for _, id := range itemIDs {
		deleteItem(db, id)
	}
}

func insertLedger(db *sql.DB, ledger types.Ledger) {
	query := fmt.Sprintf(
		"INSERT INTO ledger ("+
			"id, expense_id, lender_user_id, borrower_user_id, share"+
			") VALUES ('%s', '%s', '%s', '%s', '%s');",
		ledger.ID, ledger.ExpenseID, ledger.LenderUserID, ledger.BorrowerUesrID, ledger.Share,
	)
	db.Exec(query)
}

func deleteLedger(db *sql.DB, ledgerID uuid.UUID) {
	query := fmt.Sprintf("DELETE FROM ledger WHERE id='%s';", ledgerID)
	db.Exec(query)
}

func deleteLedgers(db *sql.DB, ledgerIDs []uuid.UUID) {
	for _, id := range ledgerIDs {
		deleteLedger(db, id)
	}
}

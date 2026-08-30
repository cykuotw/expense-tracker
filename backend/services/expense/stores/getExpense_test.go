package store_test

import (
	"expense-tracker/backend/config"
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGetExpenseByID(t *testing.T) {
	// prepare test data
	db := openTestDB(t)
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
		SplitRule:      "Equally",
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
	originalPageSize := config.Envs.ExpensesPerPage
	config.Envs.ExpensesPerPage = 25
	t.Cleanup(func() {
		config.Envs.ExpensesPerPage = originalPageSize
	})

	// prepare test data
	db := openTestDB(t)
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
			ExpenseTime:    t,
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
		order              types.ExpenseListOrder
		status             types.ExpenseListStatus
		totalPage          int64
		expectFail         bool
		expectExpenseCount []int
		expectExpenseID    [][]uuid.UUID
		expectHasMore      []bool
		expectError        []error
	}

	newestFirstIDs := make([]uuid.UUID, len(idList))
	for i := range idList {
		newestFirstIDs[i] = idList[len(idList)-1-i]
	}

	subtests := []testcase{
		{
			name:               "valid newest first",
			groupID:            mockGroupID.String(),
			order:              types.ExpenseListOrderNewest,
			status:             types.ExpenseListStatusUnsettled,
			totalPage:          4,
			expectFail:         false,
			expectExpenseCount: []int{25, 25, 10, 0},
			expectExpenseID: [][]uuid.UUID{
				newestFirstIDs[:25],
				newestFirstIDs[25:50],
				newestFirstIDs[50:60],
				nil,
			},
			expectHasMore: []bool{true, true, false},
			expectError:   []error{nil, nil, nil, types.ErrNoRemainingExpenses},
		},
		{
			name:               "valid oldest first",
			groupID:            mockGroupID.String(),
			order:              types.ExpenseListOrderOldest,
			status:             types.ExpenseListStatusUnsettled,
			totalPage:          4,
			expectFail:         false,
			expectExpenseCount: []int{25, 25, 10, 0},
			expectExpenseID: [][]uuid.UUID{
				idList[:25],
				idList[25:50],
				idList[50:60],
				nil,
			},
			expectHasMore: []bool{true, true, false},
			expectError:   []error{nil, nil, nil, types.ErrNoRemainingExpenses},
		},
		{
			name:               "invalid group id",
			groupID:            uuid.NewString(),
			order:              types.ExpenseListOrderNewest,
			status:             types.ExpenseListStatusUnsettled,
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
				expensePage, err := store.GetExpenseList(test.groupID, page, test.order, test.status)

				if test.expectFail {
					assert.Nil(t, expensePage)
					assert.Equal(t, test.expectError[0], err)
				} else {
					if err == nil {
						assert.Equal(t, test.expectExpenseCount[page], len(expensePage.Expenses))
						assert.Equal(t, test.expectHasMore[page], expensePage.HasMore)
						for i, exp := range expensePage.Expenses {
							assert.Equal(t, test.expectExpenseID[page][i], exp.ID)
						}
					} else {
						assert.Equal(t, test.expectError[page], err)
					}
				}
			}
		})
	}
}

func TestGetExpenseListFiltersBySettlementStatus(t *testing.T) {
	originalPageSize := config.Envs.ExpensesPerPage
	config.Envs.ExpensesPerPage = 25
	t.Cleanup(func() {
		config.Envs.ExpensesPerPage = originalPageSize
	})

	db := openTestDB(t)
	store := expense.NewStore(db)
	now := time.Now()
	expenses := []types.Expense{
		{
			ID:             uuid.New(),
			Description:    "unsettled",
			GroupID:        mockGroupID,
			CreateByUserID: mockCreatorID,
			PayByUserId:    mockPayerID,
			ExpenseTypeID:  mockExpenseTypeID,
			CreateTime:     now,
			ExpenseTime:    now,
			IsSettled:      false,
			Total:          decimal.NewFromInt(10),
			Currency:       "CAD",
			SplitRule:      "Equally",
		},
		{
			ID:             uuid.New(),
			Description:    "settled",
			GroupID:        mockGroupID,
			CreateByUserID: mockCreatorID,
			PayByUserId:    mockPayerID,
			ExpenseTypeID:  mockExpenseTypeID,
			CreateTime:     now.Add(time.Minute),
			ExpenseTime:    now.Add(time.Minute),
			IsSettled:      true,
			Total:          decimal.NewFromInt(20),
			Currency:       "CAD",
			SplitRule:      "Equally",
		},
	}
	ids := make([]uuid.UUID, 0, len(expenses))
	for _, exp := range expenses {
		assert.NoError(t, insertExpense(db, exp))
		ids = append(ids, exp.ID)
	}
	t.Cleanup(func() { deleteExpenses(db, ids) })

	for _, test := range []struct {
		name   string
		status types.ExpenseListStatus
		wantID uuid.UUID
	}{
		{name: "unsettled", status: types.ExpenseListStatusUnsettled, wantID: expenses[0].ID},
		{name: "settled", status: types.ExpenseListStatusSettled, wantID: expenses[1].ID},
	} {
		t.Run(test.name, func(t *testing.T) {
			page, err := store.GetExpenseList(mockGroupID.String(), 0, types.ExpenseListOrderNewest, test.status)
			if !assert.NoError(t, err) {
				return
			}

			assert.False(t, page.HasMore)
			if assert.Len(t, page.Expenses, 1) {
				assert.Equal(t, test.wantID, page.Expenses[0].ID)
			}
		})
	}
}

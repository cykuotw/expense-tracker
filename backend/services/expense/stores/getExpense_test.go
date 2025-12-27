package store_test

import (
	"expense-tracker/backend/config"
	"expense-tracker/backend/db"
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

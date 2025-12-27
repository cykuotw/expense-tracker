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

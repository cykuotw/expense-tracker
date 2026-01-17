package store_test

import (
	"database/sql"
	expense "expense-tracker/backend/services/expense/stores"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

func TestCreateBalanceLedger(t *testing.T) {
	// prepare test data
	db := openTestDB(t)
	store := expense.NewStore(db)

	type testcase struct {
		name           string
		mockBalanceIds []uuid.UUID
		mocLedgerIds   []uuid.UUID
		expectFail     bool
		expectError    error
	}

	subtests := []testcase{
		{
			name: "valid",
			mockBalanceIds: []uuid.UUID{
				uuid.New(),
				uuid.New(),
			},
			mocLedgerIds: []uuid.UUID{
				uuid.New(),
				uuid.New(),
			},
			expectFail:  false,
			expectError: nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			err := store.CreateBalanceLedger(test.mockBalanceIds, test.mocLedgerIds)
			defer deleteBalanceLedger(db, test.mockBalanceIds, test.mocLedgerIds)

			if (err != nil) != test.expectFail {
				t.Errorf("expected fail: %v but got %v", test.expectFail, err)
			}
		})
	}
}

func deleteBalanceLedger(db *sql.DB, balanceIds []uuid.UUID, ledgerIds []uuid.UUID) {
	for _, balanceId := range balanceIds {
		for _, ledgerId := range ledgerIds {
			query := fmt.Sprintf(`
				DELETE FROM balance_ledger
				WHERE balance_id = '%s' AND ledger_id = '%s';
			`, balanceId.String(), ledgerId.String())

			db.Exec(query)
		}
	}
}

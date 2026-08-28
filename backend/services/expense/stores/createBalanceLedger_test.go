package store_test

import (
	"database/sql"
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
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
			balances := make([]*types.Balance, 0, len(test.mockBalanceIds))
			for _, balanceID := range test.mockBalanceIds {
				balance := &types.Balance{ID: balanceID, SenderUserID: uuid.New(), ReceiverUserID: uuid.New(), GroupID: uuid.New(), Share: decimal.NewFromInt(1)}
				require.NoError(t, ensureBalanceParents(db, balance))
				_, err := db.Exec(`INSERT INTO balance (id, sender_user_id, receiver_user_id, share, group_id, create_time_utc, is_outdated, is_settled) VALUES ($1, $2, $3, $4, $5, now(), FALSE, FALSE)`, balance.ID, balance.SenderUserID, balance.ReceiverUserID, balance.Share, balance.GroupID)
				require.NoError(t, err)
				balances = append(balances, balance)
			}
			defer deleteBalances(db, balances)

			for _, ledgerID := range test.mocLedgerIds {
				parent := newTestExpense(uuid.New())
				require.NoError(t, insertExpense(db, parent))
				defer deleteExpense(db, parent.ID)
				_, err := db.Exec(`INSERT INTO ledger (id, expense_id, lender_user_id, borrower_user_id, share) VALUES ($1, $2, $3, $4, 1)`, ledgerID, parent.ID, parent.CreateByUserID, parent.PayByUserId)
				require.NoError(t, err)
			}
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

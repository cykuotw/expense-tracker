package expense

import (
	"testing"

	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestReconcileGroupSettlement(t *testing.T) {
	lenderID := uuid.New()
	borrowerID := uuid.New()
	ledger := &types.Ledger{
		ID:             uuid.New(),
		LenderUserID:   lenderID,
		BorrowerUesrID: borrowerID,
		Share:          decimal.NewFromInt(12),
	}
	matchingBalance := types.Balance{
		SenderUserID:   borrowerID,
		ReceiverUserID: lenderID,
		Share:          decimal.NewFromInt(12),
	}

	tests := []struct {
		name                  string
		ledgers               []*types.Ledger
		balances              []types.Balance
		expenseCount          int
		expensesWithoutLedger int
		linkCount             int
		wantState             SettlementReconciliationState
	}{
		{name: "committed", wantState: SettlementCommitted},
		{name: "committed with current settled history", balances: []types.Balance{{SenderUserID: borrowerID, ReceiverUserID: lenderID, Share: decimal.NewFromInt(12), IsSettled: true}}, wantState: SettlementCommitted},
		{name: "not started", ledgers: []*types.Ledger{ledger}, balances: []types.Balance{matchingBalance}, expenseCount: 1, linkCount: 1, wantState: SettlementNotStarted},
		{name: "missing current balance", ledgers: []*types.Ledger{ledger}, expenseCount: 1, wantState: SettlementInconsistent},
		{name: "expense without ledger", expenseCount: 1, expensesWithoutLedger: 1, wantState: SettlementInconsistent},
		{name: "stale current balance", balances: []types.Balance{matchingBalance}, wantState: SettlementInconsistent},
		{name: "wrong amount", ledgers: []*types.Ledger{ledger}, balances: []types.Balance{{SenderUserID: borrowerID, ReceiverUserID: lenderID, Share: decimal.NewFromInt(11)}}, expenseCount: 1, linkCount: 1, wantState: SettlementInconsistent},
		{name: "missing relationship", ledgers: []*types.Ledger{ledger}, balances: []types.Balance{matchingBalance}, expenseCount: 1, wantState: SettlementInconsistent},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &reconciliationStoreStub{
				ledgers:               test.ledgers,
				balances:              test.balances,
				expenseCount:          test.expenseCount,
				expensesWithoutLedger: test.expensesWithoutLedger,
				linkCount:             test.linkCount,
			}
			result, err := ReconcileGroupSettlement(store, NewController(), uuid.NewString())
			require.NoError(t, err)
			require.Equal(t, test.wantState, result.State)
			require.Equal(t, len(test.ledgers), result.UnsettledLedgerCount)
			require.Equal(t, len(test.balances), result.CurrentBalanceCount)
		})
	}
}

type reconciliationStoreStub struct {
	ledgers               []*types.Ledger
	balances              []types.Balance
	expenseCount          int
	expensesWithoutLedger int
	linkCount             int
}

func (s *reconciliationStoreStub) GetLedgerUnsettledFromGroup(string) ([]*types.Ledger, error) {
	return s.ledgers, nil
}

func (s *reconciliationStoreStub) GetCurrentBalancesByGroupID(string) ([]types.Balance, error) {
	return s.balances, nil
}

func (s *reconciliationStoreStub) GetUnsettledExpenseReconciliationCounts(string) (int, int, error) {
	return s.expenseCount, s.expensesWithoutLedger, nil
}

func (s *reconciliationStoreStub) CountCurrentBalanceLedgerLinks(string) (int, error) {
	return s.linkCount, nil
}

package expense

import (
	"cmp"
	"slices"

	"expense-tracker/backend/types"
)

type SettlementReconciliationState string

const (
	SettlementNotStarted   SettlementReconciliationState = "not_started"
	SettlementCommitted    SettlementReconciliationState = "committed"
	SettlementInconsistent SettlementReconciliationState = "inconsistent"
)

type SettlementReconciliation struct {
	State                         SettlementReconciliationState `json:"state"`
	UnsettledExpenseCount         int                           `json:"unsettledExpenseCount"`
	UnsettledLedgerCount          int                           `json:"unsettledLedgerCount"`
	ExpensesWithoutLedger         int                           `json:"expensesWithoutLedger"`
	ExpectedBalanceCount          int                           `json:"expectedBalanceCount"`
	CurrentBalanceCount           int                           `json:"currentBalanceCount"`
	OpenBalanceCount              int                           `json:"openBalanceCount"`
	CurrentBalanceLedgerLinkCount int                           `json:"currentBalanceLedgerLinkCount"`
}

type SettlementReconciliationStore interface {
	GetLedgerUnsettledFromGroup(groupID string) ([]*types.Ledger, error)
	GetCurrentBalancesByGroupID(groupID string) ([]types.Balance, error)
	GetUnsettledExpenseReconciliationCounts(groupID string) (expenseCount, expensesWithoutLedger int, err error)
	CountCurrentBalanceLedgerLinks(groupID string) (int, error)
}

// ReconcileGroupSettlement performs no writes. Callers that need a stable
// accounting snapshot should invoke it inside the group accounting lock.
func ReconcileGroupSettlement(store SettlementReconciliationStore, controller types.ExpenseController, groupID string) (SettlementReconciliation, error) {
	ledgers, err := store.GetLedgerUnsettledFromGroup(groupID)
	if err != nil {
		return SettlementReconciliation{}, err
	}
	currentBalances, err := store.GetCurrentBalancesByGroupID(groupID)
	if err != nil {
		return SettlementReconciliation{}, err
	}
	expenseCount, expensesWithoutLedger, err := store.GetUnsettledExpenseReconciliationCounts(groupID)
	if err != nil {
		return SettlementReconciliation{}, err
	}
	linkCount, err := store.CountCurrentBalanceLedgerLinks(groupID)
	if err != nil {
		return SettlementReconciliation{}, err
	}
	expectedBalances := controller.DebtSimplify(ledgers)

	result := SettlementReconciliation{
		UnsettledExpenseCount:         expenseCount,
		UnsettledLedgerCount:          len(ledgers),
		ExpensesWithoutLedger:         expensesWithoutLedger,
		ExpectedBalanceCount:          len(expectedBalances),
		CurrentBalanceCount:           len(currentBalances),
		CurrentBalanceLedgerLinkCount: linkCount,
	}
	for _, balance := range currentBalances {
		if !balance.IsSettled {
			result.OpenBalanceCount++
		}
	}
	if expenseCount == 0 {
		if len(ledgers) == 0 && result.OpenBalanceCount == 0 {
			result.State = SettlementCommitted
		} else {
			result.State = SettlementInconsistent
		}
		return result, nil
	}
	if expensesWithoutLedger == 0 &&
		balancesMatch(expectedBalances, currentBalances) &&
		linkCount == len(currentBalances)*len(ledgers) {
		result.State = SettlementNotStarted
		return result, nil
	}
	result.State = SettlementInconsistent
	return result, nil
}

func balancesMatch(expected []*types.Balance, current []types.Balance) bool {
	if len(expected) != len(current) {
		return false
	}
	expectedCopy := slices.Clone(expected)
	currentCopy := slices.Clone(current)
	slices.SortFunc(expectedCopy, compareBalancePointers)
	slices.SortFunc(currentCopy, compareBalances)
	for index := range expectedCopy {
		if expectedCopy[index].SenderUserID != currentCopy[index].SenderUserID ||
			expectedCopy[index].ReceiverUserID != currentCopy[index].ReceiverUserID ||
			!expectedCopy[index].Share.Equal(currentCopy[index].Share) {
			return false
		}
	}
	return true
}

func compareBalancePointers(left, right *types.Balance) int {
	return compareBalances(*left, *right)
}

func compareBalances(left, right types.Balance) int {
	if order := cmp.Compare(left.SenderUserID.String(), right.SenderUserID.String()); order != 0 {
		return order
	}
	if order := cmp.Compare(left.ReceiverUserID.String(), right.ReceiverUserID.String()); order != 0 {
		return order
	}
	return left.Share.Cmp(right.Share)
}

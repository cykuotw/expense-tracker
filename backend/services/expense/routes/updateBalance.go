package expense

import (
	"fmt"

	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

type balanceRebuildStore interface {
	GetLedgerUnsettledFromGroup(groupID string) ([]*types.Ledger, error)
	OutdateBalanceByGroupId(groupID string) error
	CreateBalances(groupID string, balances []*types.Balance) error
	CreateBalanceLedger(balanceIDs []uuid.UUID, ledgerIDs []uuid.UUID) error
}

type balanceRebuildError struct {
	stage string
	err   error
}

func (e *balanceRebuildError) Error() string {
	return fmt.Sprintf("balance rebuild %s: %v", e.stage, e.err)
}

func (e *balanceRebuildError) Unwrap() error {
	return e.err
}

func (h *Handler) updateBalanceWithStore(store balanceRebuildStore, groupId string) error {
	// get unsettled ledgers
	ledgers, err := store.GetLedgerUnsettledFromGroup(groupId)
	if err != nil {
		return &balanceRebuildError{stage: "ledger_read", err: err}
	}
	ledgerIds := []uuid.UUID{}
	for _, ledger := range ledgers {
		ledgerIds = append(ledgerIds, ledger.ID)
	}

	// outdate previous non-settled balances
	err = store.OutdateBalanceByGroupId(groupId)
	if err != nil {
		return &balanceRebuildError{stage: "balance_outdate", err: err}
	}

	// create balances
	balances := h.controller.DebtSimplify(ledgers)
	balanceIds := []uuid.UUID{}
	for i := 0; i < len(balances); i++ {
		balances[i].ID = uuid.New()
		balanceIds = append(balanceIds, balances[i].ID)
	}
	err = store.CreateBalances(groupId, balances)
	if err != nil {
		return &balanceRebuildError{stage: "balance_create", err: err}
	}

	// create balance_ledger
	err = store.CreateBalanceLedger(balanceIds, ledgerIds)
	if err != nil {
		return &balanceRebuildError{stage: "balance_ledger_create", err: err}
	}

	return nil
}

package expense

import (
	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

func (h *Handler) updateBalance(groupId string) error {
	return h.updateBalanceWithStore(h.store, groupId)
}

func (h *Handler) updateBalanceWithStore(store types.ExpenseStore, groupId string) error {
	// get unsettled ledgers
	ledgers, err := store.GetLedgerUnsettledFromGroup(groupId)
	if err != nil {
		return err
	}
	ledgerIds := []uuid.UUID{}
	for _, ledger := range ledgers {
		ledgerIds = append(ledgerIds, ledger.ID)
	}

	// outdate previous non-settled balances
	err = store.OutdateBalanceByGroupId(groupId)
	if err != nil {
		return err
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
		return err
	}

	// create balance_ledger
	err = store.CreateBalanceLedger(balanceIds, ledgerIds)
	if err != nil {
		return err
	}

	return nil
}

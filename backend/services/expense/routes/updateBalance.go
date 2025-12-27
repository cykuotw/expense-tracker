package expense

import (
	"github.com/google/uuid"
)

func (h *Handler) updateBalance(groupId string) error {
	// get unsettled ledgers
	ledgers, err := h.store.GetLedgerUnsettledFromGroup(groupId)
	if err != nil {
		return err
	}
	ledgerIds := []uuid.UUID{}
	for _, ledger := range ledgers {
		ledgerIds = append(ledgerIds, ledger.ID)
	}

	// outdate previous non-settled balances
	err = h.store.OutdateBalanceByGroupId(groupId)
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
	err = h.store.CreateBalances(groupId, balances)
	if err != nil {
		return err
	}

	// create balance_ledger
	err = h.store.CreateBalanceLedger(balanceIds, ledgerIds)
	if err != nil {
		return err
	}

	return nil
}

package expense

import (
	"fmt"
	"slices"

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
	// outdate previous non-settled balances
	err = store.OutdateBalanceByGroupId(groupId)
	if err != nil {
		return &balanceRebuildError{stage: "balance_outdate", err: err}
	}

	ledgersByCurrency := make(map[string][]*types.Ledger)
	for _, ledger := range ledgers {
		ledgersByCurrency[ledger.Currency] = append(ledgersByCurrency[ledger.Currency], ledger)
	}
	currencies := make([]string, 0, len(ledgersByCurrency))
	for currency := range ledgersByCurrency {
		currencies = append(currencies, currency)
	}
	slices.Sort(currencies)
	if len(currencies) == 0 {
		if err := store.CreateBalances(groupId, nil); err != nil {
			return &balanceRebuildError{stage: "balance_create", err: err}
		}
		if err := store.CreateBalanceLedger(nil, nil); err != nil {
			return &balanceRebuildError{stage: "balance_ledger_create", err: err}
		}
		return nil
	}

	for _, currency := range currencies {
		currencyLedgers := ledgersByCurrency[currency]
		ledgerIDs := make([]uuid.UUID, 0, len(currencyLedgers))
		for _, ledger := range currencyLedgers {
			ledgerIDs = append(ledgerIDs, ledger.ID)
		}
		balances := h.controller.DebtSimplify(currencyLedgers)
		balanceIDs := make([]uuid.UUID, 0, len(balances))
		for _, balance := range balances {
			balance.ID = uuid.New()
			balance.Currency = currency
			balanceIDs = append(balanceIDs, balance.ID)
		}
		if err := store.CreateBalances(groupId, balances); err != nil {
			return &balanceRebuildError{stage: "balance_create", err: err}
		}
		if err := store.CreateBalanceLedger(balanceIDs, ledgerIDs); err != nil {
			return &balanceRebuildError{stage: "balance_ledger_create", err: err}
		}
	}

	return nil
}

package expense

import (
	"expense-tracker/backend/types"
)

type usernameBatchStore interface {
	GetUsernamesByIDs(userIDs []string) (map[string]string, error)
}

type ledgerBatchStore interface {
	GetLedgersByExpenseIDs(expenseIDs []string) (map[string][]*types.Ledger, error)
}

func usernamesByIDs(store types.UserStore, userIDs []string) (map[string]string, error) {
	if batchStore, ok := store.(usernameBatchStore); ok {
		return batchStore.GetUsernamesByIDs(userIDs)
	}

	usernames := make(map[string]string, len(userIDs))
	for _, userID := range userIDs {
		if _, found := usernames[userID]; found {
			continue
		}
		username, err := store.GetUsernameByID(userID)
		if err != nil {
			return nil, err
		}
		usernames[userID] = username
	}
	return usernames, nil
}

func ledgersByExpenseIDs(store types.ExpenseStore, expenseIDs []string) (map[string][]*types.Ledger, error) {
	if batchStore, ok := store.(ledgerBatchStore); ok {
		return batchStore.GetLedgersByExpenseIDs(expenseIDs)
	}

	ledgersByExpense := make(map[string][]*types.Ledger, len(expenseIDs))
	for _, expenseID := range expenseIDs {
		ledgers, err := store.GetLedgersByExpenseID(expenseID)
		if err != nil {
			return nil, err
		}
		ledgersByExpense[expenseID] = ledgers
	}
	return ledgersByExpense, nil
}

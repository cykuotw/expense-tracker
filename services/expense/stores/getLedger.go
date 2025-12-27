package store

import (
	"expense-tracker/types"
)

func (s *Store) GetLedgersByExpenseID(expenseID string) ([]*types.Ledger, error) {
	query := "SELECT * FROM ledger WHERE expense_id = $1 ORDER BY borrower_user_id ASC;"
	rows, err := s.db.Query(query, expenseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ledgerList []*types.Ledger
	for rows.Next() {
		ledger, err := scanRowIntoLedger(rows)
		if err != nil {
			return nil, err
		}
		ledgerList = append(ledgerList, ledger)
	}

	if len(ledgerList) == 0 {
		return nil, types.ErrExpenseNotExist
	}

	return ledgerList, nil
}

func (s *Store) GetLedgerUnsettledFromGroup(groupID string) ([]*types.Ledger, error) {
	query := "SELECT l.* " +
		"FROM expense AS e " +
		"JOIN ledger AS l " +
		"ON l.expense_id = e.id " +
		"WHERE e.is_settled = false AND e.is_deleted = false AND e.group_id = $1;"
	rows, err := s.db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ledgerList := []*types.Ledger{}
	for rows.Next() {
		ledger, err := scanRowIntoLedger(rows)
		if err != nil {
			return nil, err
		}
		ledgerList = append(ledgerList, ledger)
	}

	return ledgerList, nil
}

package store

func (s *Store) GetUnsettledExpenseReconciliationCounts(groupID string) (int, int, error) {
	rows, err := s.db.Query(`
		SELECT COUNT(*), COUNT(*) FILTER (
			WHERE NOT EXISTS (SELECT 1 FROM ledger WHERE ledger.expense_id = expense.id)
		)
		FROM expense
		WHERE group_id = $1 AND is_settled = FALSE AND is_deleted = FALSE;
	`, groupID)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, 0, err
		}
		return 0, 0, nil
	}
	var expenseCount int
	var expensesWithoutLedger int
	if err := rows.Scan(&expenseCount, &expensesWithoutLedger); err != nil {
		return 0, 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	return expenseCount, expensesWithoutLedger, nil
}

func (s *Store) CountCurrentBalanceLedgerLinks(groupID string) (int, error) {
	rows, err := s.db.Query(`
		SELECT COUNT(*)
		FROM balance_ledger
		JOIN balance ON balance.id = balance_ledger.balance_id
		WHERE balance.group_id = $1 AND balance.is_outdated = FALSE;
	`, groupID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, nil
	}
	var count int
	if err := rows.Scan(&count); err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

package store

import (
	"expense-tracker/types"
)

func (s *Store) UpdateLedger(ledger types.Ledger) error {
	query := "UPDATE ledger SET " +
		"expense_id = ?, " +
		"lender_user_id = ?, " +
		"borrower_user_id = ?, " +
		"share = ? " +
		"WHERE id = ?;"
	_, err := s.db.Exec(query,
		ledger.ExpenseID, ledger.LenderUserID, ledger.BorrowerUesrID,
		ledger.Share, ledger.ID)
	if err != nil {
		return err
	}
	return nil
}

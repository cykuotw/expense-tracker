package store

import (
	"expense-tracker/types"
)

func (s *Store) CreateLedger(ledger types.Ledger) error {
	query := "INSERT INTO ledger (" +
		"id, expense_id, lender_user_id, borrower_user_id, share" +
		") VALUES (?, ?, ?, ?, ?);"

	_, err := s.db.Exec(query,
		ledger.ID, ledger.ExpenseID, ledger.LenderUserID,
		ledger.BorrowerUesrID, ledger.Share.String())
	if err != nil {
		return err
	}

	return nil
}

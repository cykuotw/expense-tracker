package store

import (
	"expense-tracker/types"
)

func (s *Store) UpdateLedger(ledger types.Ledger) error {
	query := "UPDATE ledger SET " +
		"expense_id = $1, " +
		"lender_user_id = $2, " +
		"borrower_user_id = $3, " +
		"share = $4 " +
		"WHERE id = $5;"
	_, err := s.db.Exec(query,
		ledger.ExpenseID, ledger.LenderUserID, ledger.BorrowerUesrID,
		ledger.Share, ledger.ID)
	if err != nil {
		return err
	}
	return nil
}

package store

import (
	"expense-tracker/backend/types"
)

func (s *Store) UpdateLedger(ledger types.Ledger) error {
	query := "UPDATE ledger SET " +
		"lender_user_id = $1, " +
		"borrower_user_id = $2, " +
		"share = $3 " +
		"WHERE id = $4 AND expense_id = $5;"
	result, err := s.db.Exec(query,
		ledger.LenderUserID, ledger.BorrowerUesrID, ledger.Share, ledger.ID, ledger.ExpenseID)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return types.ErrLedgerNotExist
	}
	return nil
}

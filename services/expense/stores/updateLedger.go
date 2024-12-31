package store

import (
	"expense-tracker/types"
	"fmt"
)

func (s *Store) UpdateLedger(ledger types.Ledger) error {
	query := fmt.Sprintf(
		"UPDATE ledger SET "+
			"expense_id = '%s', "+
			"lender_user_id = '%s', "+
			"borrower_user_id = '%s', "+
			"share = '%s' "+
			"WHERE id = '%s';",
		ledger.ExpenseID, ledger.LenderUserID, ledger.BorrowerUesrID,
		ledger.Share, ledger.ID,
	)
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

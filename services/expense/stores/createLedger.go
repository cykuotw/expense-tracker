package store

import (
	"expense-tracker/types"
	"fmt"
)

func (s *Store) CreateLedger(ledger types.Ledger) error {
	query := fmt.Sprintf(
		"INSERT INTO ledger ("+
			"id, expense_id, lender_user_id, borrower_user_id, share"+
			") VALUES ('%s', '%s', '%s', '%s', '%s');",
		ledger.ID, ledger.ExpenseID, ledger.LenderUserID,
		ledger.BorrowerUesrID, ledger.Share.String(),
	)

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

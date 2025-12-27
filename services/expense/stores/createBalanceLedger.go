package store

import (
	"github.com/google/uuid"
)

func (s *Store) CreateBalanceLedger(balanceIds []uuid.UUID, ledgerIds []uuid.UUID) error {
	for _, balanceId := range balanceIds {
		for _, ledgerId := range ledgerIds {
			query := `
				INSERT INTO balance_ledger (
					balance_id, ledger_id
				) VALUES ($1, $2)`

			_, err := s.db.Exec(query, balanceId.String(), ledgerId.String())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

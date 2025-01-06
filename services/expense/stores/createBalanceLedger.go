package store

import (
	"fmt"

	"github.com/google/uuid"
)

func (s *Store) CreateBalanceLedger(balanceIds []uuid.UUID, ledgerIds []uuid.UUID) error {
	for _, balanceId := range balanceIds {
		for _, ledgerId := range ledgerIds {
			query := fmt.Sprintf(`
				INSERT INTO balance_ledger (
					balance_id, ledger_id
				) VALUES ('%s', '%s')
			`, balanceId.String(), ledgerId.String())

			_, err := s.db.Exec(query)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

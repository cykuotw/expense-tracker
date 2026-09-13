package store

import (
	"errors"

	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const balanceLedgerUniqueConstraint = "balance_ledger_balance_id_ledger_id_unique"

func (s *Store) CreateBalanceLedger(balanceIds []uuid.UUID, ledgerIds []uuid.UUID) error {
	for _, balanceId := range balanceIds {
		for _, ledgerId := range ledgerIds {
			query := `
				INSERT INTO balance_ledger (
					balance_id, ledger_id
				) VALUES ($1, $2)`

			_, err := s.db.Exec(query, balanceId.String(), ledgerId.String())
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == balanceLedgerUniqueConstraint {
					return types.ErrBalanceLedgerConflict
				}
				return err
			}
		}
	}

	return nil
}

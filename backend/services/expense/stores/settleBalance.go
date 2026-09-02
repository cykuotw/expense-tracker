package store

import (
	"expense-tracker/backend/types"
	"time"
)

func (s *Store) SettleBalanceByBalanceId(groupID string, balanceID string) error {
	settleTime := time.Now().UTC()
	query := "UPDATE balance SET is_settled = true, update_time_utc = $1, settle_time_utc = $1 WHERE id = $2 AND group_id = $3;"

	result, err := s.db.Exec(query, settleTime, balanceID, groupID)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return types.ErrBalanceNotExist
	}

	return nil
}

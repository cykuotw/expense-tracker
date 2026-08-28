package store

import (
	"expense-tracker/backend/types"
	"time"
)

func (s *Store) SettleBalanceByBalanceId(groupID string, balanceID string) error {
	settleTime := time.Now().UTC().Format("2006-01-02 15:04:05-0700")
	query := "UPDATE balance SET is_settled = true, settle_time_utc = $1 WHERE id = $2 AND group_id = $3;"

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

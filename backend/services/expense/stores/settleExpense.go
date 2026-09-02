package store

import (
	"time"
)

func (s *Store) SettleExpenseByGroupId(groupId string) error {
	settleTime := time.Now().UTC()
	query := "UPDATE expense SET is_settled = true, update_time_utc = $1, settle_time_utc = $1 WHERE group_id = $2 AND is_settled = false;"

	_, err := s.db.Exec(query, settleTime, groupId)
	if err != nil {
		return err
	}

	return nil
}

package store

import (
	"time"
)

func (s *Store) SettleExpenseByGroupId(groupId string) error {
	settleTime := time.Now().UTC().Format("2006-01-02 15:04:05-0700")
	query := "UPDATE expense SET is_settled = true, settle_time_utc = $1 WHERE group_id = $2;"

	_, err := s.db.Exec(query, settleTime, groupId)
	if err != nil {
		return err
	}

	return nil
}

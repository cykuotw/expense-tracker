package store

import (
	"time"
)

func (s *Store) OutdateBalanceByGroupId(groupId string) error {
	updateTime := time.Now().UTC()
	query := "UPDATE balance SET is_outdated = true, update_time_utc = $1 WHERE group_id = $2"

	_, err := s.db.Exec(query, updateTime, groupId)
	if err != nil {
		return err
	}

	return nil
}

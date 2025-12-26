package store

import (
	"time"
)

func (s *Store) OutdateBalanceByGroupId(groupId string) error {
	updateTime := time.Now().UTC().Format("2006-01-02 15:04:05-0700")
	query := "UPDATE balance SET is_outdated = true, update_time_utc = ? WHERE group_id = ?"

	_, err := s.db.Exec(query, updateTime, groupId)
	if err != nil {
		return err
	}

	return nil
}

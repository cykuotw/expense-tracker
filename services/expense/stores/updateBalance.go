package store

import (
	"fmt"
	"time"
)

func (s *Store) OutdateBalanceByGroupId(groupId string) error {
	updateTime := time.Now().UTC().Format("2006-01-02 15:04:05-0700")
	query := fmt.Sprintf(
		"UPDATE balance "+
			"SET is_outdated = true, update_time_utc = '%s' "+
			"WHERE group_id = '%s'",
		updateTime, groupId,
	)

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

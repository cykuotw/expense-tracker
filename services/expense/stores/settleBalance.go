package store

import (
	"fmt"
	"time"
)

func (s *Store) SettleBalanceByBalanceId(balanceId string) error {
	settleTime := time.Now().UTC().Format("2006-01-02 15:04:05-0700")
	query := fmt.Sprintf(
		"UPDATE balance "+
			"SET is_settled = true, settle_time_utc = '%s' "+
			"WHERE id = '%s';",
		settleTime, balanceId,
	)

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

package store

import (
	"expense-tracker/types"
	"time"
)

func (s *Store) GetBalanceByGroupId(groupId string) ([]types.Balance, error) {
	query := `
		SELECT * FROM balance
		WHERE group_id = ? AND is_outdated = FALSE AND is_settled = FALSE;
	`
	rows, err := s.db.Query(query, groupId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []types.Balance
	for rows.Next() {
		var bal types.Balance
		updateTime := new(time.Time)
		settledTime := new(time.Time)
		err := rows.Scan(
			&bal.ID,
			&bal.SenderUserID,
			&bal.ReceiverUserID,
			&bal.Share,
			&bal.GroupID,
			&bal.CreateTime,
			&bal.IsOutdated,
			&updateTime,
			&bal.IsSettled,
			&settledTime,
		)
		if err != nil {
			return nil, err
		}
		if updateTime != nil && !updateTime.IsZero() {
			bal.UpdateTime = *updateTime
		}
		if settledTime != nil && !settledTime.IsZero() {
			bal.SettledTime = *settledTime
		}

		balances = append(balances, bal)
	}

	return balances, nil
}

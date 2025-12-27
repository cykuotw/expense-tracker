package store

import (
	"expense-tracker/types"
)

func (s *Store) CreateBalances(groupId string, balances []*types.Balance) error {
	for _, balance := range balances {
		query := "INSERT INTO balance (" +
			"id, " +
			"sender_user_id, receiver_user_id, share, " +
			"group_id " +
			") VALUES ($1, $2, $3, $4, $5)"

		_, err := s.db.Exec(query,
			balance.ID,
			balance.SenderUserID, balance.ReceiverUserID, balance.Share.String(),
			groupId,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

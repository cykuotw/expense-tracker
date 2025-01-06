package store

import (
	"expense-tracker/types"
	"fmt"
)

func (s *Store) CreateBalances(groupId string, balances []*types.Balance) error {
	for _, balance := range balances {
		query := fmt.Sprintf(
			"INSERT INTO balance ("+
				"id, "+
				"sender_user_id, receiver_user_id, share, "+
				"group_id "+
				") VALUES ('%s', '%s', '%s', '%s', '%s')",
			balance.ID,
			balance.SenderUserID, balance.ReceiverUserID, balance.Share.String(),
			groupId,
		)

		_, err := s.db.Exec(query)
		if err != nil {
			return err
		}
	}

	return nil
}

package group

import (
	"expense-tracker/backend/types"

	"github.com/shopspring/decimal"
)

func (s *Store) GetGroupCardBalanceSummary(groupID string, userID string) (types.GroupBalanceStatus, decimal.Decimal, error) {
	query := `
		SELECT sender_user_id, receiver_user_id, share
		FROM balance
		WHERE group_id = $1 AND is_outdated = FALSE AND is_settled = FALSE;
	`

	rows, err := s.db.Query(query, groupID)
	if err != nil {
		return types.GroupBalanceStatusSettled, decimal.Zero, err
	}
	defer rows.Close()

	net := decimal.Zero
	for rows.Next() {
		var senderUserID string
		var receiverUserID string
		var share decimal.Decimal

		if err := rows.Scan(&senderUserID, &receiverUserID, &share); err != nil {
			return types.GroupBalanceStatusSettled, decimal.Zero, err
		}

		switch userID {
		case receiverUserID:
			net = net.Add(share)
		case senderUserID:
			net = net.Sub(share)
		}
	}

	if err := rows.Err(); err != nil {
		return types.GroupBalanceStatusSettled, decimal.Zero, err
	}

	if net.IsZero() {
		return types.GroupBalanceStatusSettled, decimal.Zero, nil
	}
	if net.GreaterThan(decimal.Zero) {
		return types.GroupBalanceStatusOwed, net, nil
	}
	return types.GroupBalanceStatusOwing, net.Abs(), nil
}

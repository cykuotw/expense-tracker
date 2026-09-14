package store

import (
	"expense-tracker/backend/types"
	"time"

	"github.com/google/uuid"
)

func (s *Store) SettleBalanceByBalanceID(groupID string, balanceID string, actorID uuid.UUID) error {
	rows, err := s.db.Query(`SELECT sender_user_id, receiver_user_id, is_settled
		FROM balance
		WHERE id = $1 AND group_id = $2 AND is_outdated = FALSE
		FOR UPDATE`, balanceID, groupID)
	if err != nil {
		return err
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		_ = rows.Close()
		return types.ErrBalanceNotExist
	}

	var senderID uuid.UUID
	var receiverID uuid.UUID
	var settled bool
	if err := rows.Scan(&senderID, &receiverID, &settled); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	if actorID != senderID && actorID != receiverID {
		return types.ErrUserNotPermitted
	}
	if settled {
		return nil
	}

	settleTime := time.Now().UTC()
	query := "UPDATE balance SET is_settled = true, update_time_utc = $1, settle_time_utc = $1 WHERE id = $2 AND group_id = $3 AND is_outdated = false AND is_settled = false;"

	result, err := s.db.Exec(query, settleTime, balanceID, groupID)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return types.ErrBalanceNotExist
	}

	return nil
}

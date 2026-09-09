package store

import (
	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

func (s *Store) LockGroupCurrency(groupID string) (string, error) {
	rows, err := s.db.Query(`SELECT btrim(currency) FROM groups WHERE id = $1 FOR UPDATE`, groupID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return "", err
		}
		return "", types.ErrGroupNotExist
	}

	var currency string
	if err := rows.Scan(&currency); err != nil {
		return "", err
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return currency, nil
}

func (s *Store) GetCurrencyAmountDigits(currency string) (int32, error) {
	rows, err := s.db.Query(`SELECT amount_digits FROM currency WHERE code = $1`, currency)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, types.ErrUnsupportedCurrency
	}

	var digits int32
	if err := rows.Scan(&digits); err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return digits, nil
}

func (s *Store) CheckGroupParticipants(groupID string, userIDs []uuid.UUID) error {
	rows, err := s.db.Query(`SELECT user_id FROM group_member WHERE group_id = $1`, groupID)
	if err != nil {
		return err
	}
	defer rows.Close()

	members := make(map[uuid.UUID]struct{})
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return err
		}
		members[userID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, userID := range userIDs {
		if _, exists := members[userID]; !exists {
			return types.ErrGroupParticipantNotAllowed
		}
	}
	return nil
}

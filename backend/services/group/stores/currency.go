package group

import (
	"database/sql"
	"errors"
	"expense-tracker/backend/types"
)

func (s *Store) GetGroupCurrency(groupID string) (string, error) {
	query := "SELECT currency FROM groups WHERE id = $1;"
	var currency string
	if err := s.db.QueryRow(query, groupID).Scan(&currency); errors.Is(err, sql.ErrNoRows) {
		return "", types.ErrGroupNotExist
	} else if err != nil {
		return "", err
	}

	return currency, nil
}

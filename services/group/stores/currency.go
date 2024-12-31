package group

import (
	"expense-tracker/types"
	"fmt"
)

func (s *Store) GetGroupCurrency(groupID string) (string, error) {
	query := fmt.Sprintf("SELECT currency FROM groups WHERE id='%s';", groupID)
	rows, err := s.db.Query(query)
	if err != nil {
		return "", nil
	}
	defer rows.Close()

	currency := ""
	for rows.Next() {
		err := rows.Scan(&currency)
		if err != nil {
			return "", nil
		}
	}

	if currency == "" {
		return "", types.ErrGroupNotExist
	}

	return currency, nil
}

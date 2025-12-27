package group

import (
	"expense-tracker/types"
)

func (s *Store) GetGroupCurrency(groupID string) (string, error) {
	query := "SELECT currency FROM groups WHERE id = $1;"
	rows, err := s.db.Query(query, groupID)
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

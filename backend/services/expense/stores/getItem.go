package store

import (
	"expense-tracker/backend/types"
)

func (s *Store) GetItemsByExpenseID(expenseID string) ([]*types.Item, error) {
	query := "SELECT * FROM item WHERE expense_id = $1 ORDER BY id;"
	rows, err := s.db.Query(query, expenseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	itemList := make([]*types.Item, 0)
	for rows.Next() {
		item := new(types.Item)
		item, err := scanRowIntoItem(rows)
		if err != nil {
			return nil, err
		}
		itemList = append(itemList, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return itemList, nil
}

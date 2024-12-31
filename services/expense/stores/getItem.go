package store

import (
	"expense-tracker/types"
	"fmt"
)

func (s *Store) GetItemsByExpenseID(expenseID string) ([]*types.Item, error) {
	query := fmt.Sprintf("SELECT * FROM item WHERE expense_id='%s' ORDER BY id;", expenseID)
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var itemList []*types.Item
	for rows.Next() {
		item := new(types.Item)
		item, err := scanRowIntoItem(rows)
		if err != nil {
			return nil, err
		}
		itemList = append(itemList, item)
	}

	if len(itemList) == 0 {
		return nil, types.ErrExpenseNotExist
	}

	return itemList, nil
}

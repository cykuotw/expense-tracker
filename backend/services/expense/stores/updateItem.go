package store

import (
	"expense-tracker/backend/types"
)

func (s *Store) UpdateItem(item types.Item) error {
	query := "UPDATE item SET " +
		"name = $1, " +
		"amount = $2, " +
		"unit = $3, " +
		"unit_price = $4 " +
		"WHERE id = $5 AND expense_id = $6;"
	result, err := s.db.Exec(query,
		item.Name, item.Amount, item.Unit, item.UnitPrice, item.ID, item.ExpenseID)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return types.ErrItemNotExist
	}
	return nil
}

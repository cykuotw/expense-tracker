package store

import (
	"expense-tracker/types"
)

func (s *Store) UpdateItem(item types.Item) error {
	query := "UPDATE item SET " +
		"expense_id = $1, " +
		"name = $2, " +
		"amount = $3, " +
		"unit = $4, " +
		"unit_price = $5 " +
		"WHERE id = $6;"
	_, err := s.db.Exec(query,
		item.ExpenseID, item.Name, item.Amount, item.Unit,
		item.UnitPrice, item.ID)
	if err != nil {
		return err
	}
	return nil
}

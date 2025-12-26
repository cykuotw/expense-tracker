package store

import (
	"expense-tracker/types"
)

func (s *Store) UpdateItem(item types.Item) error {
	query := "UPDATE item SET " +
		"expense_id = ?, " +
		"name = ?, " +
		"amount = ?, " +
		"unit = ?, " +
		"unit_price = ? " +
		"WHERE id = ?;"
	_, err := s.db.Exec(query,
		item.ExpenseID, item.Name, item.Amount, item.Unit,
		item.UnitPrice, item.ID)
	if err != nil {
		return err
	}
	return nil
}

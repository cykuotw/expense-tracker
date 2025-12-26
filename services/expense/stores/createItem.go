package store

import (
	"expense-tracker/types"
)

func (s *Store) CreateItem(item types.Item) error {
	query := "INSERT INTO item (" +
		"id, expense_id, name, amount, " +
		"unit, unit_price" +
		") VALUES (?, ?, ?, ?, ?, ?);"

	_, err := s.db.Exec(query,
		item.ID, item.ExpenseID, item.Name, item.Amount.String(),
		item.Unit, item.UnitPrice.String())
	if err != nil {
		return err
	}

	return nil
}

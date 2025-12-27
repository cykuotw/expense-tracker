package store

import (
	"expense-tracker/backend/types"
)

func (s *Store) CreateItem(item types.Item) error {
	query := "INSERT INTO item (" +
		"id, expense_id, name, amount, " +
		"unit, unit_price" +
		") VALUES ($1, $2, $3, $4, $5, $6);"

	_, err := s.db.Exec(query,
		item.ID, item.ExpenseID, item.Name, item.Amount.String(),
		item.Unit, item.UnitPrice.String())
	if err != nil {
		return err
	}

	return nil
}

package store

import (
	"expense-tracker/backend/types"
)

func (s *Store) CreateItem(item types.Item) error {
	query := "INSERT INTO item (" +
		"id, expense_id, name, amount, unit, unit_price, " +
		"description, quantity, confirmed_unit, confirmed_unit_price, line_total, position" +
		") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);"

	legacyUnit := ""
	if item.Unit != nil {
		legacyUnit = *item.Unit
	}

	_, err := s.db.Exec(query,
		item.ID, item.ExpenseID, item.Description, "1", legacyUnit, item.LineTotal.String(),
		item.Description, item.Quantity, item.Unit, item.UnitPrice, item.LineTotal, item.Position)
	if err != nil {
		return err
	}

	return nil
}

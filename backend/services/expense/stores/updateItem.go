package store

import (
	"expense-tracker/backend/types"
)

func (s *Store) UpdateItem(item types.Item) error {
	legacyUnit := ""
	if item.Unit != nil {
		legacyUnit = *item.Unit
	}
	query := "UPDATE item SET " +
		"name = $1, " +
		"amount = $2, " +
		"unit = $3, " +
		"unit_price = $4, " +
		"description = $5, " +
		"quantity = $6, " +
		"confirmed_unit = $7, " +
		"confirmed_unit_price = $8, " +
		"line_total = $9, " +
		"position = $10 " +
		"WHERE id = $11 AND expense_id = $12;"
	result, err := s.db.Exec(query,
		item.Description, "1", legacyUnit, item.LineTotal,
		item.Description, item.Quantity, item.Unit, item.UnitPrice, item.LineTotal, item.Position,
		item.ID, item.ExpenseID)
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

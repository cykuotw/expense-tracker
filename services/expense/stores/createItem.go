package store

import (
	"expense-tracker/types"
	"fmt"
)

func (s *Store) CreateItem(item types.Item) error {
	query := fmt.Sprintf(
		"INSERT INTO item ("+
			"id, expense_id, name, amount, "+
			"unit, unit_price"+
			") VALUES ('%s', '%s', '%s', '%s', '%s', '%s');",
		item.ID, item.ExpenseID, item.Name, item.Amount.String(),
		item.Unit, item.UnitPrice.String(),
	)

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

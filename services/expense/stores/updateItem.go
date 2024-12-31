package store

import (
	"expense-tracker/types"
	"fmt"
)

func (s *Store) UpdateItem(item types.Item) error {
	query := fmt.Sprintf(
		"UPDATE item SET "+
			"expense_id = '%s', "+
			"name = '%s', "+
			"amount = '%s', "+
			"unit = '%s', "+
			"unit_price = '%s' "+
			"WHERE id = '%s';",
		item.ExpenseID, item.Name, item.Amount, item.Unit,
		item.UnitPrice, item.ID,
	)
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

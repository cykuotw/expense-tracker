package store

import (
	"expense-tracker/types"
	"fmt"
)

func (s *Store) DeleteExpense(expense types.Expense) error {
	query := fmt.Sprintf(
		"UPDATE expense SET is_deleted = true "+
			"WHERE id = '%s';",
		expense.ID,
	)
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

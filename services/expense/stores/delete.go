package store

import (
	"expense-tracker/types"
	"fmt"
	"time"
)

func (s *Store) DeleteExpense(expense types.Expense) error {
	deleteTime := time.Now().UTC().Format("2006-01-02 15:04:05-0700")
	query := fmt.Sprintf(
		"UPDATE expense SET is_deleted = true, delete_time_utc = '%s' "+
			"WHERE id = '%s';",
		deleteTime, expense.ID,
	)
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

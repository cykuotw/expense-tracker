package store

import (
	"expense-tracker/backend/types"
	"time"
)

func (s *Store) DeleteExpense(expense types.Expense) error {
	deleteTime := time.Now().UTC()
	query := "UPDATE expense SET is_deleted = true, update_time_utc = $1, delete_time_utc = $1 WHERE id = $2 AND is_deleted = false;"
	_, err := s.db.Exec(query, deleteTime, expense.ID)
	if err != nil {
		return err
	}

	return nil
}

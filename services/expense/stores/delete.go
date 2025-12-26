package store

import (
	"expense-tracker/types"
	"time"
)

func (s *Store) DeleteExpense(expense types.Expense) error {
	deleteTime := time.Now().UTC().Format("2006-01-02 15:04:05-0700")
	query := "UPDATE expense SET is_deleted = true, delete_time_utc = ? WHERE id = ?;"
	_, err := s.db.Exec(query, deleteTime, expense.ID)
	if err != nil {
		return err
	}

	return nil
}

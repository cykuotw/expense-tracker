package store

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func (s *Store) ClearItemPositions(expenseID uuid.UUID) error {
	_, err := s.db.Exec("UPDATE item SET position = NULL WHERE expense_id = $1", expenseID)
	return err
}

func (s *Store) DeleteItemsNotIn(expenseID uuid.UUID, itemIDs []uuid.UUID) error {
	if len(itemIDs) == 0 {
		_, err := s.db.Exec("DELETE FROM item WHERE expense_id = $1", expenseID)
		return err
	}

	args := make([]any, 0, len(itemIDs)+1)
	args = append(args, expenseID)
	placeholders := make([]string, 0, len(itemIDs))
	for index, itemID := range itemIDs {
		args = append(args, itemID)
		placeholders = append(placeholders, fmt.Sprintf("$%d", index+2))
	}
	query := "DELETE FROM item WHERE expense_id = $1 AND id NOT IN (" + strings.Join(placeholders, ", ") + ")"
	_, err := s.db.Exec(query, args...)
	return err
}

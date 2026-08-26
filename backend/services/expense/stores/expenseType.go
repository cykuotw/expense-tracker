package store

import (
	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

func (s *Store) GetExpenseType() ([]*types.ExpenseType, error) {
	query := "SELECT id, name, category FROM expense_type ORDER BY category, name;"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenseTypes []*types.ExpenseType
	for rows.Next() {
		expenseType := new(types.ExpenseType)
		if err := rows.Scan(&expenseType.ID, &expenseType.Name, &expenseType.Category); err != nil {
			return nil, err
		}
		expenseTypes = append(expenseTypes, expenseType)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return expenseTypes, nil
}

func (s *Store) GetExpenseTypeById(id uuid.UUID) (string, error) {
	query := "SELECT name FROM expense_type WHERE id = $1;"

	rows, err := s.db.Query(query, id.String())
	if err != nil {
		return "", err
	}
	defer rows.Close()

	name := ""
	for rows.Next() {
		if err := rows.Scan(&name); err != nil {
			return "", err
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	return name, nil
}

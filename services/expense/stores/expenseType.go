package store

import (
	"expense-tracker/types"

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
		rows.Scan(&expenseType.ID, &expenseType.Name, &expenseType.Category)
		expenseTypes = append(expenseTypes, expenseType)
	}

	return expenseTypes, nil
}

func (s *Store) GetExpenseTypeById(id uuid.UUID) (string, error) {
	query := "SELECT name FROM expense_type WHERE id = ?;"

	rows, err := s.db.Query(query, id.String())
	if err != nil {
		return "", err
	}
	defer rows.Close()

	name := ""
	for rows.Next() {
		rows.Scan(&name)
	}

	return name, nil
}

package store

import (
	"expense-tracker/backend/types"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func (s *Store) GetCategoryBreakdownsByExpenseIDs(expenseIDs []string) (map[string][]types.ExpenseCategoryAmount, error) {
	result := make(map[string][]types.ExpenseCategoryAmount)
	if len(expenseIDs) == 0 {
		return result, nil
	}
	rows, err := s.db.Query(`SELECT b.expense_id, b.expense_type_id, b.amount, b.note, t.name, t.category
        FROM expense_category_breakdown b JOIN expense_type t ON t.id = b.expense_type_id
        WHERE b.expense_id = ANY($1::uuid[]) ORDER BY b.expense_id, b.position`, pq.Array(expenseIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var row types.ExpenseCategoryAmount
		if err := rows.Scan(&id, &row.ExpenseTypeID, &row.Amount, &row.Note, &row.ExpenseType, &row.Category); err != nil {
			return nil, err
		}
		result[id] = append(result[id], row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// The caller holds the active-group lock and validates precision and the total.
func (s *Store) ReplaceExpenseCategoryBreakdown(expenseID uuid.UUID, rows []types.ExpenseCategoryAmount) error {
	if _, err := s.db.Exec(`DELETE FROM expense_category_breakdown WHERE expense_id = $1`, expenseID); err != nil {
		return err
	}
	for position, row := range rows {
		result, err := s.db.Exec(`INSERT INTO expense_category_breakdown (expense_id, position, expense_type_id, amount, note)
            SELECT $1, $2, id, $4, $5 FROM expense_type WHERE id = $3`, expenseID, position, row.ExpenseTypeID, row.Amount.String(), row.Note)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return types.ErrInvalidCategoryBreakdown
		}
	}
	return nil
}

package store

import (
	"expense-tracker/config"
	"expense-tracker/types"

	"github.com/google/uuid"
)

func (s *Store) GetExpenseByID(expenseID string) (*types.Expense, error) {
	query := "SELECT * FROM expense WHERE id = ?;"
	rows, err := s.db.Query(query, expenseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expense := new(types.Expense)
	for rows.Next() {
		expense, err = scanRowIntoExpense(rows)
		if err != nil {
			return nil, err
		}
	}

	if expense.ID == uuid.Nil {
		return nil, types.ErrExpenseNotExist
	}

	return expense, nil
}

func (s *Store) GetExpenseList(groupID string, page int64) ([]*types.Expense, error) {
	offset := page * config.Envs.ExpensesPerPage
	limit := config.Envs.ExpensesPerPage

	query := "SELECT * FROM expense " +
		"WHERE group_id = ? AND is_deleted = False " +
		"ORDER BY create_time_utc DESC " +
		"OFFSET ? LIMIT ?;"

	rows, err := s.db.Query(query, groupID, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenseList []*types.Expense
	for rows.Next() {
		expense := new(types.Expense)
		expense, err = scanRowIntoExpense(rows)
		if err != nil {
			return nil, err
		}
		expenseList = append(expenseList, expense)
	}

	if len(expenseList) == 0 {
		return nil, types.ErrNoRemainingExpenses
	}

	return expenseList, nil
}

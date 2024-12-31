package store

import (
	"expense-tracker/config"
	"expense-tracker/types"
	"fmt"

	"github.com/google/uuid"
)

func (s *Store) GetExpenseByID(expenseID string) (*types.Expense, error) {
	query := fmt.Sprintf("SELECT * FROM expense WHERE id='%s';", expenseID)
	rows, err := s.db.Query(query)
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

	query := fmt.Sprintf(
		"SELECT * FROM expense "+
			"WHERE group_id = '%s' AND is_deleted = False "+
			"ORDER BY create_time_utc ASC "+
			"OFFSET '%d' LIMIT '%d';",
		groupID, offset, limit,
	)

	rows, err := s.db.Query(query)
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

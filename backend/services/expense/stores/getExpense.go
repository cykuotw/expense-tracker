package store

import (
	"expense-tracker/backend/config"
	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

func (s *Store) GetExpenseByID(expenseID string) (*types.Expense, error) {
	query := "SELECT " + expenseSelectColumns + " FROM expense WHERE id = $1;"
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if expense.ID == uuid.Nil {
		return nil, types.ErrExpenseNotExist
	}

	return expense, nil
}

func (s *Store) GetExpenseList(groupID string, page int64, order types.ExpenseListOrder, status types.ExpenseListStatus) (*types.ExpenseListPage, error) {
	offset := page * config.Envs.ExpensesPerPage
	limit := config.Envs.ExpensesPerPage

	query := "SELECT " + expenseSelectColumns + " FROM expense " +
		"WHERE group_id = $1 AND is_deleted = False " +
		"ORDER BY COALESCE(occurred_on, (expense_time_utc AT TIME ZONE 'UTC')::date) DESC, expense_time_utc DESC, id DESC " +
		"OFFSET $2 LIMIT $3;"
	if order == types.ExpenseListOrderOldest {
		query = "SELECT " + expenseSelectColumns + " FROM expense " +
			"WHERE group_id = $1 AND is_deleted = False " +
			"ORDER BY COALESCE(occurred_on, (expense_time_utc AT TIME ZONE 'UTC')::date) ASC, expense_time_utc ASC, id ASC " +
			"OFFSET $2 LIMIT $3;"
	}
	if status == types.ExpenseListStatusUnsettled || status == types.ExpenseListStatusSettled {
		settled := status == types.ExpenseListStatusSettled
		query = "SELECT " + expenseSelectColumns + " FROM expense " +
			"WHERE group_id = $1 AND is_deleted = False AND is_settled = $2 " +
			"ORDER BY COALESCE(occurred_on, (expense_time_utc AT TIME ZONE 'UTC')::date) DESC, expense_time_utc DESC, id DESC " +
			"OFFSET $3 LIMIT $4;"
		if order == types.ExpenseListOrderOldest {
			query = "SELECT " + expenseSelectColumns + " FROM expense " +
				"WHERE group_id = $1 AND is_deleted = False AND is_settled = $2 " +
				"ORDER BY COALESCE(occurred_on, (expense_time_utc AT TIME ZONE 'UTC')::date) ASC, expense_time_utc ASC, id ASC " +
				"OFFSET $3 LIMIT $4;"
		}

		return s.getExpenseList(query, groupID, settled, offset, limit)
	}

	return s.getExpenseList(query, groupID, offset, limit)
}

func (s *Store) getExpenseList(query string, args ...any) (*types.ExpenseListPage, error) {
	limit := config.Envs.ExpensesPerPage
	args[len(args)-1] = limit + 1

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expenseList := make([]*types.Expense, 0, limit+1)
	for rows.Next() {
		expense := new(types.Expense)
		expense, err = scanRowIntoExpense(rows)
		if err != nil {
			return nil, err
		}
		expenseList = append(expenseList, expense)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(expenseList) == 0 {
		return nil, types.ErrNoRemainingExpenses
	}

	hasMore := len(expenseList) > int(limit)
	if hasMore {
		expenseList = expenseList[:limit]
	}

	return &types.ExpenseListPage{Expenses: expenseList, HasMore: hasMore}, nil
}

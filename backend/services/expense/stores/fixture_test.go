package store_test

import (
	"database/sql"
	"expense-tracker/backend/types"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func newTestExpense(expenseID uuid.UUID) types.Expense {
	return types.Expense{
		ID:             expenseID,
		Description:    "test expense",
		GroupID:        uuid.New(),
		CreateByUserID: uuid.New(),
		PayByUserId:    uuid.New(),
		ExpenseTypeID:  uuid.New(),
		CreateTime:     time.Now().UTC(),
		Total:          decimal.NewFromInt(1),
		Currency:       "CAD",
		SplitRule:      "Equally",
		OccurredOn:     "2026-08-31",
	}
}

func ensureExpenseParents(db *sql.DB, expense types.Expense) error {
	if err := ensureUser(db, expense.CreateByUserID); err != nil {
		return err
	}
	if err := ensureUser(db, expense.PayByUserId); err != nil {
		return err
	}
	if err := ensureGroup(db, expense.GroupID, expense.CreateByUserID); err != nil {
		return err
	}
	if err := ensureMember(db, expense.GroupID, expense.PayByUserId); err != nil {
		return err
	}
	_, err := db.Exec(`INSERT INTO expense_type (id, name, category) VALUES ($1, $2, 'test') ON CONFLICT (id) DO NOTHING`, expense.ExpenseTypeID, "test-"+expense.ExpenseTypeID.String()[:8])
	return err
}

func ensureExpense(db *sql.DB, expenseID uuid.UUID) error {
	expense := newTestExpense(expenseID)
	if err := ensureExpenseParents(db, expense); err != nil {
		return err
	}

	_, err := db.Exec(`INSERT INTO expense (
		id, description, group_id, create_by_user_id, pay_by_user_id, exp_type_id,
		is_settled, sub_total, tax_fee_tip, total, currency, create_time_utc, split_rule
	) VALUES (
		$1, $2, $3, $4, $5, $6, FALSE, 0, 0, $7, $8, $9, $10
	) ON CONFLICT (id) DO NOTHING`,
		expense.ID, expense.Description, expense.GroupID, expense.CreateByUserID,
		expense.PayByUserId, expense.ExpenseTypeID, expense.Total, expense.Currency,
		expense.CreateTime, expense.SplitRule)
	return err
}

func ensureBalanceParents(db *sql.DB, balance *types.Balance) error {
	if err := ensureUser(db, balance.SenderUserID); err != nil {
		return err
	}
	if err := ensureUser(db, balance.ReceiverUserID); err != nil {
		return err
	}
	if err := ensureGroup(db, balance.GroupID, balance.SenderUserID); err != nil {
		return err
	}
	return ensureMember(db, balance.GroupID, balance.ReceiverUserID)
}

func ensureUser(db *sql.DB, userID uuid.UUID) error {
	shortID := userID.String()[:8]
	_, err := db.Exec(`INSERT INTO users (id, username, firstname, lastname, email, password_hash, create_time_utc, is_active, has_local_password, role)
		VALUES ($1, $2, 'Test', 'User', $3, 'not-used', $4, TRUE, TRUE, 'user') ON CONFLICT (id) DO NOTHING`,
		userID, "test-"+shortID, shortID+"@example.test", time.Now().UTC())
	return err
}

func ensureGroup(db *sql.DB, groupID uuid.UUID, creatorID uuid.UUID) error {
	_, err := db.Exec(`INSERT INTO groups (id, group_name, description, create_time_utc, is_active, create_by_user_id, currency)
		VALUES ($1, $2, '', $3, TRUE, $4, 'CAD') ON CONFLICT (id) DO NOTHING`,
		groupID, "test-"+groupID.String()[:8], time.Now().UTC(), creatorID)
	if err != nil {
		return err
	}
	return ensureMember(db, groupID, creatorID)
}

func ensureMember(db *sql.DB, groupID uuid.UUID, userID uuid.UUID) error {
	_, err := db.Exec(`INSERT INTO group_member (id, group_id, user_id) VALUES ($1, $2, $3) ON CONFLICT (group_id, user_id) DO NOTHING`, uuid.New(), groupID, userID)
	return err
}

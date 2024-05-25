package expense

import (
	"database/sql"
	"expense-tracker/types"
	"fmt"
	"time"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateExpense(expense types.Expense) error {
	createTime := time.Now().UTC().Format("2006-01-02 15:04:05-0700")
	query := fmt.Sprintf(
		"INSERT INTO expense ("+
			"id, description, group_id, "+
			"create_by_user_id, pay_by_user_id, provider_name, "+
			"exp_type_id, is_settled, "+
			"sub_total, tax_fee_tip, total, "+
			"currency, invoice_pic_url, create_time_utc"+
			") VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%t', "+
			"'%s', '%s', '%s', '%s', '%s', '%s')",
		expense.ID, expense.Description, expense.GroupID,
		expense.CreateByUserID, expense.PayByUserId, expense.ProviderName,
		expense.ExpenseTypeID, false,
		expense.SubTotal.String(), expense.TaxFeeTip.String(), expense.Total.String(),
		expense.Currency, expense.InvoicePicUrl, createTime,
	)

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreateItem(item types.Item) error {
	return nil
}

func (s *Store) CreateLedger(ledger types.Ledger) error {
	return nil
}

func (s *Store) GetExpenseByID(expenseID string) (*types.Expense, error) {
	return nil, nil
}

func (s *Store) GetExpenseList(page int64) ([]*types.Expense, error) {
	return nil, nil
}

func (s *Store) GetItemsByExpenseID(expenseID string) ([]*types.Item, error) {
	return nil, nil
}

func (s *Store) GetLedgersByExpenseID(expenseID string) ([]*types.Ledger, error) {
	return nil, nil
}

func (s *Store) GetLedgerUnsettledFromGroup(groupID string) ([]*types.Ledger, error) {
	return nil, nil
}

func (s *Store) UpdateExpenseSettleInGroup(groupID string) error {
	return nil
}

func (s *Store) UpdateExpense(expense types.Expense) error {
	return nil
}

func (s *Store) UpdateItem(item types.Item) error {
	return nil
}

func (s *Store) UpdateLedger(ledger types.Ledger) error {
	return nil
}

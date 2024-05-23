package expense

import (
	"database/sql"
	"expense-tracker/types"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateExpense(expense types.Expense) error {
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

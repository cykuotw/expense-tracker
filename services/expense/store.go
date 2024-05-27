package expense

import (
	"database/sql"
	"expense-tracker/config"
	"expense-tracker/types"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	query := fmt.Sprintf(
		"INSERT INTO item ("+
			"id, expense_id, name, amount, "+
			"unit, unit_price"+
			") VALUES ('%s', '%s', '%s', '%s', '%s', '%s');",
		item.ID, item.ExpenseID, item.Name, item.Amount.String(),
		item.Unit, item.UnitPrice.String(),
	)

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreateLedger(ledger types.Ledger) error {
	query := fmt.Sprintf(
		"INSERT INTO ledger ("+
			"id, expense_id, lender_user_id, borrower_user_id, share"+
			") VALUES ('%s', '%s', '%s', '%s', '%s');",
		ledger.ID, ledger.ExpenseID, ledger.LenderUserID,
		ledger.BorrowerUesrID, ledger.Share.String(),
	)

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

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
			"WHERE group_id = '%s' "+
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

func (s *Store) GetItemsByExpenseID(expenseID string) ([]*types.Item, error) {
	query := fmt.Sprintf("SELECT * FROM item WHERE expense_id='%s' ORDER BY id;", expenseID)
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var itemList []*types.Item
	for rows.Next() {
		item := new(types.Item)
		item, err := scanRowIntoItem(rows)
		if err != nil {
			return nil, err
		}
		itemList = append(itemList, item)
	}

	if len(itemList) == 0 {
		return nil, types.ErrExpenseNotExist
	}

	return itemList, nil
}

func (s *Store) GetLedgersByExpenseID(expenseID string) ([]*types.Ledger, error) {
	query := fmt.Sprintf(
		"SELECT * FROM ledger WHERE expense_id='%s';", expenseID,
	)
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ledgerList []*types.Ledger
	for rows.Next() {
		ledger, err := scanRowIntoLedger(rows)
		if err != nil {
			return nil, err
		}
		ledgerList = append(ledgerList, ledger)
	}

	if len(ledgerList) == 0 {
		return nil, types.ErrExpenseNotExist
	}

	return ledgerList, nil
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

func scanRowIntoExpense(rows *sql.Rows) (*types.Expense, error) {
	expense := new(types.Expense)

	err := rows.Scan(
		&expense.ID,
		&expense.Description,
		&expense.GroupID,
		&expense.CreateByUserID,
		&expense.PayByUserId,
		&expense.ProviderName,
		&expense.ExpenseTypeID,
		&expense.IsSettled,
		&expense.SubTotal,
		&expense.TaxFeeTip,
		&expense.Total,
		&expense.Currency,
		&expense.InvoicePicUrl,
		&expense.CreateTime,
	)
	if err != nil {
		return nil, err
	}
	return expense, nil
}

func scanRowIntoItem(rows *sql.Rows) (*types.Item, error) {
	item := new(types.Item)

	err := rows.Scan(
		&item.ID,
		&item.ExpenseID,
		&item.Name,
		&item.Amount,
		&item.Unit,
		&item.UnitPrice,
	)
	if err != nil {
		return nil, err
	}
	return item, err
}

func scanRowIntoLedger(rows *sql.Rows) (*types.Ledger, error) {
	ledger := new(types.Ledger)

	err := rows.Scan(
		&ledger.ID,
		&ledger.ExpenseID,
		&ledger.LenderUserID,
		&ledger.BorrowerUesrID,
		&ledger.Share,
	)
	if err != nil {
		return nil, err
	}
	return ledger, err
}

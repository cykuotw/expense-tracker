package store

import (
	"expense-tracker/types"
	"time"
)

func (s *Store) CreateExpense(expense types.Expense) error {
	createTime := time.Now().UTC().Format("2006-01-02 15:04:05-0700")
	query := "INSERT INTO expense (" +
		"id, description, group_id, " +
		"create_by_user_id, pay_by_user_id, provider_name, " +
		"exp_type_id, is_settled, " +
		"sub_total, tax_fee_tip, total, " +
		"currency, invoice_pic_url, " +
		"create_time_utc, update_time_utc, expense_time_utc, " +
		"split_rule " +
		") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)"

	_, err := s.db.Exec(query,
		expense.ID, expense.Description, expense.GroupID,
		expense.CreateByUserID, expense.PayByUserId, expense.ProviderName,
		expense.ExpenseTypeID, false,
		expense.SubTotal.String(), expense.TaxFeeTip.String(), expense.Total.String(),
		expense.Currency, expense.InvoicePicUrl, createTime, createTime, createTime,
		expense.SplitRule)
	if err != nil {
		return err
	}

	return nil
}

package store

import (
	"expense-tracker/backend/types"
)

func (s *Store) UpdateExpenseSettleInGroup(groupID string) error {
	// settle all expense with groupID
	query := "UPDATE expense SET is_settled=true WHERE group_id = $1 and is_settled=false;"
	_, err := s.db.Exec(query, groupID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) UpdateExpense(expense types.Expense) error {
	updateTime := expense.UpdateTime.UTC().Format("2006-01-02 15:04:05-0700")
	expenseTime := expense.ExpenseTime.UTC().Format("2006-01-02 15:04:05-0700")

	query := "UPDATE expense SET " +
		"description = $1, " +
		"group_id = $2, " +
		"pay_by_user_id = $3, " +
		"update_time_utc = $4, " +
		"expense_time_utc = $5, " +
		"provider_name = $6, " +
		"exp_type_id = $7, " +
		"is_settled = $8, " +
		"sub_total = $9, " +
		"tax_fee_tip = $10, " +
		"total = $11, " +
		"currency = $12, " +
		"invoice_pic_url = $13, " +
		"split_rule = $14 " +
		"WHERE id = $15;"
	_, err := s.db.Exec(query,
		expense.Description, expense.GroupID,
		expense.PayByUserId,
		updateTime, expenseTime,
		expense.ProviderName,
		expense.ExpenseTypeID, expense.IsSettled, expense.SubTotal,
		expense.TaxFeeTip, expense.Total, expense.Currency,
		expense.InvoicePicUrl, expense.SplitRule, expense.ID)
	if err != nil {
		return err
	}

	return nil
}

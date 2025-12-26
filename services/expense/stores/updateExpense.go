package store

import (
	"expense-tracker/types"
)

func (s *Store) UpdateExpenseSettleInGroup(groupID string) error {
	// settle all expense with groupID
	query := "UPDATE expense SET is_settled=true WHERE group_id = ? and is_settled=false;"
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
		"description = ?, " +
		"group_id = ?, " +
		"pay_by_user_id = ?, " +
		"update_time_utc = ?, " +
		"expense_time_utc = ?, " +
		"provider_name = ?, " +
		"exp_type_id = ?, " +
		"is_settled = ?, " +
		"sub_total = ?, " +
		"tax_fee_tip = ?, " +
		"total = ?, " +
		"currency = ?, " +
		"invoice_pic_url = ?, " +
		"split_rule = ? " +
		"WHERE id = ?;"
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

package store

import (
	"expense-tracker/types"
	"fmt"
)

func (s *Store) UpdateExpenseSettleInGroup(groupID string) error {
	// settle all expense with groupID
	query := fmt.Sprintf(
		"UPDATE expense SET is_settled=true "+
			"WHERE group_id='%s' and is_settled=false;",
		groupID,
	)
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) UpdateExpense(expense types.Expense) error {
	updateTime := expense.UpdateTime.UTC().Format("2006-01-02 15:04:05-0700")
	expenseTime := expense.ExpenseTime.UTC().Format("2006-01-02 15:04:05-0700")

	query := fmt.Sprintf(
		"UPDATE expense SET "+
			"description = '%s', "+
			"group_id = '%s', "+
			"pay_by_user_id = '%s', "+
			"update_time_utc = '%s', "+
			"expense_time_utc = '%s', "+
			"provider_name = '%s', "+
			"exp_type_id = '%s', "+
			"is_settled = '%t', "+
			"sub_total = '%s', "+
			"tax_fee_tip = '%s', "+
			"total = '%s', "+
			"currency = '%s', "+
			"invoice_pic_url = '%s', "+
			"split_rule = '%s' "+
			"WHERE id = '%s';",
		expense.Description, expense.GroupID,
		expense.PayByUserId,
		updateTime, expenseTime,
		expense.ProviderName,
		expense.ExpenseTypeID, expense.IsSettled, expense.SubTotal,
		expense.TaxFeeTip, expense.Total, expense.Currency,
		expense.InvoicePicUrl, expense.SplitRule,
		expense.ID,
	)
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

package store

import (
	"expense-tracker/backend/types"
	"time"
)

func (s *Store) UpdateExpenseSettleInGroup(groupID string) error {
	// settle all expense with groupID
	settleTime := time.Now().UTC()
	query := "UPDATE expense SET is_settled = true, update_time_utc = $1, settle_time_utc = $1 WHERE group_id = $2 AND is_settled = false;"
	_, err := s.db.Exec(query, settleTime, groupID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) UpdateExpense(expense types.Expense) error {
	updateTime := time.Now().UTC()

	query := "UPDATE expense SET " +
		"description = $1, " +
		"group_id = $2, " +
		"pay_by_user_id = $3, " +
		"update_time_utc = $4, " +
		"provider_name = $5, " +
		"exp_type_id = $6, " +
		"is_settled = $7, " +
		"sub_total = $8, " +
		"tax_fee_tip = $9, " +
		"total = $10, " +
		"currency = $11, " +
		"invoice_pic_url = $12, " +
		"split_rule = $13, " +
		"occurred_on = COALESCE(NULLIF($14, '')::date, occurred_on) " +
		"WHERE id = $15;"
	_, err := s.db.Exec(query,
		expense.Description, expense.GroupID,
		expense.PayByUserId,
		updateTime,
		expense.ProviderName,
		expense.ExpenseTypeID, expense.IsSettled, expense.SubTotal,
		expense.TaxFeeTip, expense.Total, expense.Currency,
		expense.InvoicePicUrl, expense.SplitRule, expense.OccurredOn, expense.ID)
	if err != nil {
		return err
	}

	return nil
}

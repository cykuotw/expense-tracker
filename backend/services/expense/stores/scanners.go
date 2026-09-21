package store

import (
	"database/sql"
	"time"

	"expense-tracker/backend/types"
)

const expenseSelectColumns = `
	id, description, group_id, create_by_user_id, pay_by_user_id,
	provider_name, exp_type_id, is_settled, sub_total, tax_fee_tip,
	total, currency, invoice_pic_url, create_time_utc, update_time_utc,
	expense_time_utc, allocation_mode, is_deleted, delete_time_utc,
	settle_time_utc, occurred_on`

const balanceSelectColumns = `
	id, sender_user_id, receiver_user_id, share, group_id,
	create_time_utc, is_outdated, update_time_utc, is_settled, settle_time_utc`

const itemSelectColumns = `id, expense_id, name, amount, unit, unit_price`

const ledgerSelectColumns = `id, expense_id, lender_user_id, borrower_user_id, share`

const qualifiedLedgerSelectColumns = `
	l.id, l.expense_id, l.lender_user_id, l.borrower_user_id, l.share`

func scanRowIntoExpense(rows *sql.Rows) (*types.Expense, error) {
	expense := new(types.Expense)
	updateTime := sql.NullTime{}
	settleTime := sql.NullTime{}
	deleteTime := sql.NullTime{}
	occurredOn := sql.NullTime{}

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
		&updateTime,
		&expense.ExpenseTime,
		&expense.AllocationMode,
		&expense.IsDeleted,
		&deleteTime,
		&settleTime,
		&occurredOn,
	)
	if err != nil {
		return nil, err
	}

	expense.CreateTime = expense.CreateTime.UTC()
	expense.ExpenseTime = expense.ExpenseTime.UTC()
	if occurredOn.Valid {
		expense.OccurredOn = occurredOn.Time.Format(time.DateOnly)
	} else {
		expense.OccurredOn = expense.ExpenseTime.Format(time.DateOnly)
	}
	if updateTime.Valid {
		expense.UpdateTime = updateTime.Time.UTC()
	}
	if settleTime.Valid {
		expense.SettleTime = settleTime.Time.UTC()
	}
	if deleteTime.Valid {
		expense.DeleteTime = deleteTime.Time.UTC()
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

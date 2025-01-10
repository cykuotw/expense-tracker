package store

import (
	"database/sql"
	"expense-tracker/types"
	"time"
)

func scanRowIntoExpense(rows *sql.Rows) (*types.Expense, error) {
	expense := new(types.Expense)
	updateTime := new(time.Time)
	settleTime := new(time.Time)

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
		updateTime,
		&expense.ExpenseTime,
		&expense.SplitRule,
		&expense.IsDeleted,
		&expense.DeleteTime,
		settleTime,
	)
	if err != nil {
		return nil, err
	}

	if !updateTime.IsZero() {
		expense.UpdateTime = *updateTime
	}
	if !settleTime.IsZero() {
		expense.SettleTime = *settleTime
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

package store_test

import (
	"database/sql"
	"expense-tracker/backend/types"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var mockGroupID = uuid.New()
var mockCreatorID = uuid.New()
var mockPayerID = uuid.New()
var mockExpenseTypeID = uuid.New()

const expenseTestColumns = `
	id, description, group_id, create_by_user_id, pay_by_user_id,
	provider_name, exp_type_id, is_settled, sub_total, tax_fee_tip,
	total, currency, invoice_pic_url, create_time_utc, update_time_utc,
	expense_time_utc, split_rule, is_deleted, delete_time_utc,
	settle_time_utc, occurred_on`

func selectExpense(db *sql.DB, groupID uuid.UUID) []*types.Expense {
	query := fmt.Sprintf(
		"SELECT %s FROM expense "+
			"WHERE group_id = '%s' "+
			"ORDER BY create_time_utc ASC;",
		expenseTestColumns,
		groupID,
	)
	rows, _ := db.Query(query)
	defer rows.Close()

	expList := []*types.Expense{}

	for rows.Next() {
		expense := new(types.Expense)
		updateTime := sql.NullTime{}
		settleTime := sql.NullTime{}
		deleteTime := sql.NullTime{}
		occurredOn := sql.NullTime{}

		rows.Scan(
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
			&expense.SplitRule,
			&expense.IsDeleted,
			&deleteTime,
			&settleTime,
			&occurredOn,
		)

		if updateTime.Valid {
			expense.UpdateTime = updateTime.Time
		}
		if settleTime.Valid {
			expense.SettleTime = settleTime.Time
		}
		if deleteTime.Valid {
			expense.DeleteTime = deleteTime.Time
		}
		if occurredOn.Valid {
			expense.OccurredOn = occurredOn.Time.Format(time.DateOnly)
		}
		expList = append(expList, expense)
	}

	return expList
}

func selectExpenseByID(db *sql.DB, expenseID uuid.UUID) *types.Expense {
	query := fmt.Sprintf(
		"SELECT %s FROM expense "+
			"WHERE id = '%s';",
		expenseTestColumns,
		expenseID,
	)
	rows, _ := db.Query(query)
	defer rows.Close()

	expense := new(types.Expense)

	for rows.Next() {
		updateTime := sql.NullTime{}
		settleTime := sql.NullTime{}
		deleteTime := sql.NullTime{}
		occurredOn := sql.NullTime{}

		rows.Scan(
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
			&expense.SplitRule,
			&expense.IsDeleted,
			&deleteTime,
			&settleTime,
			&occurredOn,
		)

		if updateTime.Valid {
			expense.UpdateTime = updateTime.Time
		}
		if settleTime.Valid {
			expense.SettleTime = settleTime.Time
		}
		if deleteTime.Valid {
			expense.DeleteTime = deleteTime.Time
		}
		if occurredOn.Valid {
			expense.OccurredOn = occurredOn.Time.Format(time.DateOnly)
		}
	}

	return expense
}

func insertExpense(db *sql.DB, expense types.Expense) error {
	if err := ensureExpenseParents(db, expense); err != nil {
		return err
	}
	createTime := expense.CreateTime.UTC()
	if createTime.IsZero() {
		createTime = time.Now().UTC()
	}
	expenseTime := expense.ExpenseTime
	if expenseTime.IsZero() {
		expenseTime = createTime
	}
	query := `INSERT INTO expense (
		id, description, group_id,
		create_by_user_id, pay_by_user_id, provider_name,
		exp_type_id, is_settled,
		sub_total, tax_fee_tip, total,
		currency, invoice_pic_url, create_time_utc, expense_time_utc, split_rule, occurred_on
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NULLIF($17, '')::date)`

	_, err := db.Exec(query,
		expense.ID, expense.Description, expense.GroupID,
		expense.CreateByUserID, expense.PayByUserId, expense.ProviderName,
		expense.ExpenseTypeID, expense.IsSettled,
		expense.SubTotal, expense.TaxFeeTip, expense.Total,
		expense.Currency, expense.InvoicePicUrl, createTime, expenseTime.UTC(), expense.SplitRule, expense.OccurredOn)

	return err
}

func deleteExpense(db *sql.DB, expenseId uuid.UUID) {
	query := fmt.Sprintf("DELETE FROM expense WHERE id='%s';", expenseId)
	db.Exec(query)
}

func deleteExpenses(db *sql.DB, expenseIds []uuid.UUID) {
	for _, id := range expenseIds {
		deleteExpense(db, id)
	}
}

func insertItem(db *sql.DB, item types.Item) error {
	if err := ensureExpense(db, item.ExpenseID); err != nil {
		return err
	}
	query := fmt.Sprintf(
		"INSERT INTO item ("+
			"id, expense_id, name, amount, unit, unit_price"+
			") VALUES ('%s', '%s', '%s', '%s', '%s', '%s')",
		item.ID, item.ExpenseID, item.Name, item.Amount, item.Unit, item.UnitPrice,
	)
	_, err := db.Exec(query)
	return err
}

func deleteItem(db *sql.DB, itemID uuid.UUID) {
	query := fmt.Sprintf("DELETE FROM item WHERE id='%s';", itemID)
	db.Exec(query)
}

func deleteItems(db *sql.DB, itemIDs []uuid.UUID) {
	for _, id := range itemIDs {
		deleteItem(db, id)
	}
}

func insertLedger(db *sql.DB, ledger types.Ledger) error {
	if err := ensureExpense(db, ledger.ExpenseID); err != nil {
		return err
	}
	if err := ensureUser(db, ledger.LenderUserID); err != nil {
		return err
	}
	if err := ensureUser(db, ledger.BorrowerUesrID); err != nil {
		return err
	}
	query := fmt.Sprintf(
		"INSERT INTO ledger ("+
			"id, expense_id, lender_user_id, borrower_user_id, share"+
			") VALUES ('%s', '%s', '%s', '%s', '%s');",
		ledger.ID, ledger.ExpenseID, ledger.LenderUserID, ledger.BorrowerUesrID, ledger.Share,
	)
	_, err := db.Exec(query)
	return err
}

func deleteLedger(db *sql.DB, ledgerID uuid.UUID) {
	query := fmt.Sprintf("DELETE FROM ledger WHERE id='%s';", ledgerID)
	db.Exec(query)
}

func deleteLedgers(db *sql.DB, ledgerIDs []uuid.UUID) {
	for _, id := range ledgerIDs {
		deleteLedger(db, id)
	}
}

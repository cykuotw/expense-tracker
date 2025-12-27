package store_test

import (
	"database/sql"
	"expense-tracker/backend/types"
	"fmt"

	"github.com/google/uuid"
)

var mockGroupID = uuid.New()
var mockCreatorID = uuid.New()
var mockPayerID = uuid.New()
var mockExpenseTypeID = uuid.New()

func selectExpense(db *sql.DB, groupID uuid.UUID) []*types.Expense {
	query := fmt.Sprintf(
		"SELECT * FROM expense "+
			"WHERE group_id = '%s' "+
			"ORDER BY create_time_utc ASC;",
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
		expList = append(expList, expense)
	}

	return expList
}

func selectExpenseByID(db *sql.DB, expenseID uuid.UUID) *types.Expense {
	query := fmt.Sprintf(
		"SELECT * FROM expense "+
			"WHERE id = '%s';",
		expenseID,
	)
	rows, _ := db.Query(query)
	defer rows.Close()

	expense := new(types.Expense)

	for rows.Next() {
		updateTime := sql.NullTime{}
		settleTime := sql.NullTime{}
		deleteTime := sql.NullTime{}

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
	}

	return expense
}

func insertExpense(db *sql.DB, expense types.Expense) {
	createTime := expense.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	query := fmt.Sprintf(
		"INSERT INTO expense ("+
			"id, description, group_id, "+
			"create_by_user_id, pay_by_user_id, provider_name, "+
			"exp_type_id, is_settled, "+
			"sub_total, tax_fee_tip, total, "+
			"currency, invoice_pic_url, create_time_utc, split_rule"+
			") VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%t', "+
			"'%s', '%s', '%s', '%s', '%s', '%s', '%s')",
		expense.ID, expense.Description, expense.GroupID,
		expense.CreateByUserID, expense.PayByUserId, expense.ProviderName,
		expense.ExpenseTypeID, expense.IsSettled,
		expense.SubTotal.String(), expense.TaxFeeTip.String(), expense.Total.String(),
		expense.Currency, expense.InvoicePicUrl, createTime, expense.SplitRule,
	)

	db.Exec(query)
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

func insertItem(db *sql.DB, item types.Item) {
	query := fmt.Sprintf(
		"INSERT INTO item ("+
			"id, expense_id, name, amount, unit, unit_price"+
			") VALUES ('%s', '%s', '%s', '%s', '%s', '%s')",
		item.ID, item.ExpenseID, item.Name, item.Amount, item.Unit, item.UnitPrice,
	)
	db.Exec(query)
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

func insertLedger(db *sql.DB, ledger types.Ledger) {
	query := fmt.Sprintf(
		"INSERT INTO ledger ("+
			"id, expense_id, lender_user_id, borrower_user_id, share"+
			") VALUES ('%s', '%s', '%s', '%s', '%s');",
		ledger.ID, ledger.ExpenseID, ledger.LenderUserID, ledger.BorrowerUesrID, ledger.Share,
	)
	db.Exec(query)
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

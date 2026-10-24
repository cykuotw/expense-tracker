// Package ocrtest provides isolated fixtures for OCR integration tests only.
package ocrtest

import (
	"database/sql"
	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

type Fixture struct {
	DB                                    *sql.DB
	User, Other, Group, Expense, Category uuid.UUID
}

func New(t *testing.T) Fixture {
	t.Helper()
	db, err := dbstore.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		t.Skipf("integration DB unavailable: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("integration DB unavailable: %v", err)
	}
	f := Fixture{db, uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	for _, id := range []uuid.UUID{f.User, f.Other} {
		_, err = db.Exec(`INSERT INTO users(id,username,firstname,lastname,email,password_hash,create_time_utc,is_active,has_local_password,role) VALUES($1,$2,'OCR','Test',$3,'unused',NOW(),TRUE,TRUE,'user')`, id, "ocr-"+id.String()[:20], id.String()+"@example.test")
		require.NoError(t, err)
	}
	_, err = db.Exec(`INSERT INTO groups(id,group_name,description,create_time_utc,is_active,create_by_user_id,currency,group_type) VALUES($1,'OCR test','',NOW(),TRUE,$2,'CAD','home')`, f.Group, f.User)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO group_member(id,group_id,user_id) VALUES($1,$2,$3)`, uuid.New(), f.Group, f.User)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO expense_type(id,name,category) VALUES($1,'OCR fixture','Food and Drink')`, f.Category)
	require.NoError(t, err)
	f.InsertExpense(t, f.Expense, "CAD", "Milk")
	t.Cleanup(func() {
		for _, statement := range []string{`DELETE FROM expense WHERE create_by_user_id IN ($1,$2)`, `DELETE FROM groups WHERE create_by_user_id IN ($1,$2)`, `DELETE FROM user_feature_grant WHERE user_id IN ($1,$2)`, `DELETE FROM users WHERE id IN ($1,$2)`} {
			_, err := db.Exec(statement, f.User, f.Other)
			require.NoError(t, err)
		}
		_, err := db.Exec(`DELETE FROM expense_type WHERE id=$1`, f.Category)
		require.NoError(t, err)
		db.Close()
	})
	return f
}

func (f Fixture) InsertExpense(t *testing.T, id uuid.UUID, currency, description string) {
	t.Helper()
	_, err := f.DB.Exec(`INSERT INTO group_currency(group_id,currency,enabled_for_new_expenses) VALUES($1,$2,TRUE) ON CONFLICT DO NOTHING`, f.Group, currency)
	require.NoError(t, err)
	_, err = f.DB.Exec(`INSERT INTO expense(id,description,group_id,create_by_user_id,pay_by_user_id,exp_type_id,is_settled,sub_total,tax_fee_tip,total,currency,create_time_utc,expense_time_utc,allocation_mode,is_deleted,occurred_on,provider_name) VALUES($1,'OCR test',$2,$3,$3,$4,FALSE,5.39,0,5.39,$5,NOW(),'2022-03-18 15:06:00+00','equal',FALSE,'2022-03-18','Shoppers Drug Mart')`, id, f.Group, f.User, f.Category, currency)
	require.NoError(t, err)
	_, err = f.DB.Exec(`INSERT INTO item(id,expense_id,name,amount,description,line_total,position,unit_price) VALUES($1,$2,$3,1,$3,5.39,0,5.39)`, uuid.New(), id, description)
	require.NoError(t, err)
}

package group_test

import (
	"database/sql"
	"expense-tracker/backend/config"
	"expense-tracker/backend/db"
	"testing"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	cfg := config.Envs
	conn, err := db.NewPostgreSQLStorage(cfg)
	if err != nil {
		t.Skipf("skipping: db connect error: %v", err)
	}
	err = conn.Ping()
	if err != nil {
		t.Skipf("skipping: db ping error: %v", err)
	}
	return conn
}

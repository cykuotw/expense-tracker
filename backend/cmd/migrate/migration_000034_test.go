package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration000034RejectsExistingDuplicatesWithoutChangingData(t *testing.T) {
	database := openMigrationTestDB(t)
	up, _ := readMigration000034(t)

	tx, err := database.Begin()
	require.NoError(t, err)
	defer tx.Rollback()
	createBalanceLedgerTestSchema(t, tx, "migration_34_duplicates")

	balanceID := uuid.New()
	ledgerID := uuid.New()
	_, err = tx.Exec(`INSERT INTO balance_ledger (balance_id, ledger_id) VALUES ($1, $2), ($1, $2)`, balanceID, ledgerID)
	require.NoError(t, err)
	_, err = tx.Exec("SAVEPOINT before_migration")
	require.NoError(t, err)

	_, err = tx.Exec(string(up))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve duplicate (balance_id, ledger_id) rows first")
	_, rollbackErr := tx.Exec("ROLLBACK TO SAVEPOINT before_migration")
	require.NoError(t, rollbackErr)

	var rowCount int
	require.NoError(t, tx.QueryRow(`SELECT COUNT(*) FROM balance_ledger WHERE balance_id = $1 AND ledger_id = $2`, balanceID, ledgerID).Scan(&rowCount))
	assert.Equal(t, 2, rowCount)

	var constraintCount int
	require.NoError(t, tx.QueryRow(`
		SELECT COUNT(*)
		FROM pg_constraint
		WHERE conrelid = 'balance_ledger'::regclass
		  AND conname = 'balance_ledger_balance_id_ledger_id_unique'`).Scan(&constraintCount))
	assert.Zero(t, constraintCount)
}

func TestMigration000034IsRepeatableAndReversibleForValidData(t *testing.T) {
	database := openMigrationTestDB(t)
	up, down := readMigration000034(t)

	tx, err := database.Begin()
	require.NoError(t, err)
	defer tx.Rollback()
	createBalanceLedgerTestSchema(t, tx, "migration_34_valid")

	balanceID := uuid.New()
	ledgerID := uuid.New()
	_, err = tx.Exec(`INSERT INTO balance_ledger (balance_id, ledger_id) VALUES ($1, $2)`, balanceID, ledgerID)
	require.NoError(t, err)

	_, err = tx.Exec(string(up))
	require.NoError(t, err)
	_, err = tx.Exec(string(up))
	require.NoError(t, err)

	_, err = tx.Exec("SAVEPOINT duplicate_insert")
	require.NoError(t, err)
	_, err = tx.Exec(`INSERT INTO balance_ledger (balance_id, ledger_id) VALUES ($1, $2)`, balanceID, ledgerID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "balance_ledger_balance_id_ledger_id_unique")
	_, rollbackErr := tx.Exec("ROLLBACK TO SAVEPOINT duplicate_insert")
	require.NoError(t, rollbackErr)

	var rowCount int
	require.NoError(t, tx.QueryRow(`SELECT COUNT(*) FROM balance_ledger`).Scan(&rowCount))
	assert.Equal(t, 1, rowCount)

	_, err = tx.Exec(string(down))
	require.NoError(t, err)
	_, err = tx.Exec(string(down))
	require.NoError(t, err)
	_, err = tx.Exec(`INSERT INTO balance_ledger (balance_id, ledger_id) VALUES ($1, $2)`, balanceID, ledgerID)
	require.NoError(t, err)
}

func openMigrationTestDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := dbstore.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		requireMigrationTestDB(t, "connect", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.Ping(); err != nil {
		requireMigrationTestDB(t, "ping", err)
	}
	return database
}

func requireMigrationTestDB(t *testing.T, operation string, err error) {
	t.Helper()
	if os.Getenv("EXPENSE_TRACKER_REQUIRE_TEST_DB") == "1" {
		t.Fatalf("%s required integration database: %v", operation, err)
	}
	t.Skipf("skipping: db %s error: %v", operation, err)
}

func readMigration000034(t *testing.T) ([]byte, []byte) {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrationDir := filepath.Join(filepath.Dir(sourceFile), "migrations")
	up, err := os.ReadFile(filepath.Join(migrationDir, "000034_enforce_balance_ledger_uniqueness.up.sql"))
	require.NoError(t, err)
	down, err := os.ReadFile(filepath.Join(migrationDir, "000034_enforce_balance_ledger_uniqueness.down.sql"))
	require.NoError(t, err)
	return up, down
}

func createBalanceLedgerTestSchema(t *testing.T, tx *sql.Tx, prefix string) {
	t.Helper()
	schema := strings.ReplaceAll(prefix+"_"+uuid.NewString(), "-", "_")
	_, err := tx.Exec(fmt.Sprintf(`
		CREATE SCHEMA %s;
		SET LOCAL search_path TO %s;
		CREATE TABLE balance_ledger (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			balance_id UUID NOT NULL,
			ledger_id UUID NOT NULL
		);`, schema, schema))
	require.NoError(t, err)
}

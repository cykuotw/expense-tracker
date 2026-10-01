package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMigration000039ReceiptAssociationConstraintsAndRollback(t *testing.T) {
	db := openMigrationTestDB(t)
	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Join(filepath.Dir(sourceFile), "migrations")
	up, err := os.ReadFile(filepath.Join(dir, "000039_add_expense_receipts.up.sql"))
	require.NoError(t, err)
	down, err := os.ReadFile(filepath.Join(dir, "000039_add_expense_receipts.down.sql"))
	require.NoError(t, err)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback()
	schema := strings.ReplaceAll("migration_39_"+uuid.NewString(), "-", "_")
	_, err = tx.ExecContext(t.Context(), fmt.Sprintf(`CREATE SCHEMA %s; SET LOCAL search_path TO %s, public;
        CREATE TABLE users (id UUID PRIMARY KEY);
        CREATE TABLE expense (id UUID PRIMARY KEY);`, schema, schema))
	require.NoError(t, err)
	require.NoError(t, executeSQLScript(tx, string(up)))
	accountID, expenseID := uuid.New(), uuid.New()
	_, err = tx.ExecContext(t.Context(), `INSERT INTO users(id) VALUES ($1)`, accountID)
	require.NoError(t, err)
	_, err = tx.ExecContext(t.Context(), `INSERT INTO expense(id) VALUES ($1)`, expenseID)
	require.NoError(t, err)
	insert := func(id, objectID, key uuid.UUID, role, status string) error {
		_, err := tx.ExecContext(t.Context(), `INSERT INTO expense_receipt
            (id, expense_id, account_id, object_id, temporary_key, retained_key,
            request_key, token_sha256, role, status, content_type, byte_size,
            checksum_sha256, width, height, source_expires_at)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'image/jpeg',24,$11,4,6,NOW() + INTERVAL '1 day')`,
			id, expenseID, accountID, objectID, "temporary/"+objectID.String()+".jpg",
			"retained/"+expenseID.String()+"/"+objectID.String()+".jpg", key,
			strings.Repeat("a", 64), role, status, strings.Repeat("b", 64))
		return err
	}
	require.NoError(t, insert(uuid.New(), uuid.New(), uuid.New(), "current", "pending"))
	require.NoError(t, insert(uuid.New(), uuid.New(), uuid.New(), "candidate", "pending"))
	_, err = tx.ExecContext(t.Context(), "SAVEPOINT duplicate_current")
	require.NoError(t, err)
	require.Error(t, insert(uuid.New(), uuid.New(), uuid.New(), "current", "pending"))
	_, err = tx.ExecContext(t.Context(), "ROLLBACK TO SAVEPOINT duplicate_current")
	require.NoError(t, err)
	_, err = tx.ExecContext(t.Context(), "SAVEPOINT bad_status")
	require.NoError(t, err)
	require.Error(t, insert(uuid.New(), uuid.New(), uuid.New(), "cleanup", "unknown"))
	_, err = tx.ExecContext(t.Context(), "ROLLBACK TO SAVEPOINT bad_status")
	require.NoError(t, err)
	require.NoError(t, executeSQLScript(tx, string(down)))
}

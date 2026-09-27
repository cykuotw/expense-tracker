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

func TestMigration000038BackfillsAndPreservesRollingCompatibility(t *testing.T) {
	database := openMigrationTestDB(t)
	up, down := readMigration000038(t)
	tx, err := database.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback()

	schema := strings.ReplaceAll("migration_38_"+uuid.NewString(), "-", "_")
	_, err = tx.ExecContext(t.Context(), fmt.Sprintf(`
		CREATE SCHEMA %s;
		SET LOCAL search_path TO %s, public;
		CREATE TABLE item (
			id UUID PRIMARY KEY,
			expense_id UUID NOT NULL,
			name VARCHAR(32),
			amount NUMERIC(10, 2),
			unit VARCHAR(10),
			unit_price NUMERIC(10, 3)
		);`, schema, schema))
	require.NoError(t, err)

	expenseID := uuid.New()
	firstID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	secondID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	_, err = tx.ExecContext(t.Context(), `INSERT INTO item (id, expense_id, name, amount, unit, unit_price)
		VALUES ($1, $3, 'Coffee', 2, 'cup', 3.125), ($2, $3, 'Bagel', 1, '', 4.500)`, firstID, secondID, expenseID)
	require.NoError(t, err)
	require.NoError(t, executeSQLScript(tx, string(up)))

	var description string
	var lineTotal string
	var position int
	require.NoError(t, tx.QueryRowContext(t.Context(), `
		SELECT description, line_total::text, position FROM item WHERE id = $1`, firstID,
	).Scan(&description, &lineTotal, &position))
	require.Equal(t, "Coffee", description)
	require.Equal(t, "6.250", lineTotal)
	require.Zero(t, position)

	// A pre-migration Worker can still insert using only legacy columns.
	_, err = tx.ExecContext(t.Context(), `INSERT INTO item (id, expense_id, name, amount, unit, unit_price)
		VALUES ($1, $2, 'Legacy', 1, 'each', 2.000)`, uuid.New(), expenseID)
	require.NoError(t, err)

	// A new Worker can dual-write rows that remain readable through legacy columns.
	_, err = tx.ExecContext(t.Context(), `INSERT INTO item (
		id, expense_id, name, amount, unit, unit_price,
		description, quantity, confirmed_unit, confirmed_unit_price, line_total, position
	) VALUES ($1, $2, 'New', 1, '', 7.250, 'New', NULL, NULL, NULL, 7.250, 2)`, uuid.New(), expenseID)
	require.NoError(t, err)

	_, err = tx.ExecContext(t.Context(), "SAVEPOINT duplicate_position")
	require.NoError(t, err)
	_, err = tx.ExecContext(t.Context(), `INSERT INTO item (
		id, expense_id, name, amount, unit, unit_price, description, line_total, position
	) VALUES ($1, $2, 'Duplicate', 1, '', 1, 'Duplicate', 1, 2)`, uuid.New(), expenseID)
	require.Error(t, err)
	_, rollbackErr := tx.ExecContext(t.Context(), "ROLLBACK TO SAVEPOINT duplicate_position")
	require.NoError(t, rollbackErr)

	require.NoError(t, executeSQLScript(tx, string(down)))
}

func readMigration000038(t *testing.T) ([]byte, []byte) {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrationDir := filepath.Join(filepath.Dir(sourceFile), "migrations")
	up, err := os.ReadFile(filepath.Join(migrationDir, "000038_expand_confirmed_item_contract.up.sql"))
	require.NoError(t, err)
	down, err := os.ReadFile(filepath.Join(migrationDir, "000038_expand_confirmed_item_contract.down.sql"))
	require.NoError(t, err)
	return up, down
}

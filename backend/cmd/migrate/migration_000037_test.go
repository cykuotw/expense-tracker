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

func TestMigration000037IsReversibleAndEnforcesFeatureGrantShape(t *testing.T) {
	database := openMigrationTestDB(t)
	up, down := readMigration000037(t)
	tx, err := database.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback()

	schema := strings.ReplaceAll("migration_37_"+uuid.NewString(), "-", "_")
	_, err = tx.ExecContext(t.Context(), fmt.Sprintf(`
		CREATE SCHEMA %s;
		SET LOCAL search_path TO %s, public;
		CREATE TABLE users (id UUID PRIMARY KEY);`, schema, schema))
	require.NoError(t, err)
	require.NoError(t, executeSQLScript(tx, string(up)))

	actorID := uuid.New()
	targetID := uuid.New()
	_, err = tx.ExecContext(t.Context(), "INSERT INTO users (id) VALUES ($1), ($2)", actorID, targetID)
	require.NoError(t, err)
	_, err = tx.ExecContext(t.Context(), `
		INSERT INTO user_feature_grant (user_id, feature_key, granted_by_user_id)
		VALUES ($1, 'receipt_ocr', $2)`, targetID, actorID)
	require.NoError(t, err)

	_, err = tx.ExecContext(t.Context(), "SAVEPOINT unsupported_feature")
	require.NoError(t, err)
	_, err = tx.ExecContext(t.Context(), `
		INSERT INTO user_feature_grant (user_id, feature_key, granted_by_user_id)
		VALUES ($1, 'unknown', $2)`, actorID, actorID)
	require.Error(t, err)
	_, rollbackErr := tx.ExecContext(t.Context(), "ROLLBACK TO SAVEPOINT unsupported_feature")
	require.NoError(t, rollbackErr)

	_, err = tx.ExecContext(t.Context(), "DELETE FROM users WHERE id = $1", targetID)
	require.NoError(t, err)
	var grantCount int
	require.NoError(t, tx.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM user_feature_grant WHERE user_id = $1", targetID,
	).Scan(&grantCount))
	require.Zero(t, grantCount)

	require.NoError(t, executeSQLScript(tx, string(down)))
	var tableExists bool
	require.NoError(t, tx.QueryRowContext(t.Context(),
		"SELECT to_regclass(current_schema() || '.user_feature_grant') IS NOT NULL",
	).Scan(&tableExists))
	require.False(t, tableExists)
}

func readMigration000037(t *testing.T) ([]byte, []byte) {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrationDir := filepath.Join(filepath.Dir(sourceFile), "migrations")
	up, err := os.ReadFile(filepath.Join(migrationDir, "000037_add_user_feature_grants.up.sql"))
	require.NoError(t, err)
	down, err := os.ReadFile(filepath.Join(migrationDir, "000037_add_user_feature_grants.down.sql"))
	require.NoError(t, err)
	return up, down
}

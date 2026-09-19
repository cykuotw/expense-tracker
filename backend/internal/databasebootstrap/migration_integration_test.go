package databasebootstrap

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"expense-tracker/backend/internal/migrationpolicy"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateUpAppliesTimeoutsAndRejectsDirtyState(t *testing.T) {
	database, err := dbstore.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		requireMigrationTestDatabase(t, err)
	}
	defer database.Close()
	if err := database.PingContext(t.Context()); err != nil {
		requireMigrationTestDatabase(t, err)
	}

	schema := strings.ReplaceAll("migration_runtime_"+uuid.NewString(), "-", "_")
	_, err = database.ExecContext(t.Context(), "CREATE SCHEMA "+quoteIdentifier(schema))
	require.NoError(t, err)
	defer database.ExecContext(t.Context(), "DROP SCHEMA "+quoteIdentifier(schema)+" CASCADE")

	directory := t.TempDir()
	writeMigrationFile(t, directory, "000035_baseline.up.sql", `
		CREATE TABLE migration_runtime_probe (
			id INTEGER PRIMARY KEY,
			lock_timeout TEXT NOT NULL,
			statement_timeout TEXT NOT NULL
		);
		INSERT INTO migration_runtime_probe (id, lock_timeout, statement_timeout)
		VALUES (1, current_setting('lock_timeout'), current_setting('statement_timeout'));
	`)
	writeMigrationFile(t, directory, "000035_baseline.down.sql", "DROP TABLE migration_runtime_probe;")
	writeMigrationFile(t, directory, "000036_add_note.up.sql", "ALTER TABLE migration_runtime_probe ADD COLUMN note TEXT;")
	writeMigrationFile(t, directory, "000036_add_note.down.sql", "ALTER TABLE migration_runtime_probe DROP COLUMN note;")
	writeMigrationFile(t, directory, migrationpolicy.ManifestName, `{
		"schemaVersion": 1,
		"baselineVersion": 35,
		"migrations": [{
			"version": 36,
			"name": "add_note",
			"categories": ["additive"],
			"deployment": "online",
			"backfill": {"mode": "none", "resumable": false, "notes": "No rows are rewritten."},
			"locking": {"risk": "low", "notes": "The fixture table is bounded."},
			"rollback": {"application": "compatible", "schema": "not_required", "dataLossRisk": false, "notes": "The old contract ignores the column."}
		}]
	}`)
	manifest, err := migrationpolicy.LoadDirectory(directory)
	require.NoError(t, err)

	connectionURL, err := url.Parse(dbstore.BuildPostgreSQLDSN(config.Envs))
	require.NoError(t, err)
	query := connectionURL.Query()
	query.Set("search_path", schema)
	connectionURL.RawQuery = query.Encode()
	require.NoError(t, migrateUp(directory, connectionURL.String(), manifest))
	require.NoError(t, migrateUp(directory, connectionURL.String(), manifest), "an up-to-date migration set must be idempotent")

	var lockTimeout string
	var statementTimeout string
	err = database.QueryRowContext(
		t.Context(),
		fmt.Sprintf("SELECT lock_timeout, statement_timeout FROM %s.migration_runtime_probe WHERE id = 1", quoteIdentifier(schema)),
	).Scan(&lockTimeout, &statementTimeout)
	require.NoError(t, err)
	assert.Equal(t, "5s", lockTimeout)
	assert.Contains(t, []string{"4min", "240s"}, statementTimeout)

	_, err = database.ExecContext(
		t.Context(),
		fmt.Sprintf("UPDATE %s.schema_migrations SET dirty = TRUE", quoteIdentifier(schema)),
	)
	require.NoError(t, err)
	err = migrateUp(directory, connectionURL.String(), manifest)
	require.ErrorContains(t, err, "version=000036 dirty=true")
}

func TestFailedBackfillReportsSecretSafeDirtyState(t *testing.T) {
	database, err := dbstore.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		requireMigrationTestDatabase(t, err)
	}
	defer database.Close()
	if err := database.PingContext(t.Context()); err != nil {
		requireMigrationTestDatabase(t, err)
	}

	schema := strings.ReplaceAll("migration_failure_"+uuid.NewString(), "-", "_")
	_, err = database.ExecContext(t.Context(), "CREATE SCHEMA "+quoteIdentifier(schema))
	require.NoError(t, err)
	defer database.ExecContext(t.Context(), "DROP SCHEMA "+quoteIdentifier(schema)+" CASCADE")

	directory := t.TempDir()
	writeMigrationFile(t, directory, "000035_baseline.up.sql", "CREATE TABLE migration_failure_probe (id INTEGER PRIMARY KEY);")
	writeMigrationFile(t, directory, "000035_baseline.down.sql", "DROP TABLE migration_failure_probe;")
	writeMigrationFile(t, directory, "000036_private_failure.up.sql", `
		UPDATE migration_failure_probe SET id = id;
		SELECT 1 / 0;
	`)
	writeMigrationFile(t, directory, "000036_private_failure.down.sql", "SELECT 1;")
	writeMigrationFile(t, directory, migrationpolicy.ManifestName, `{
		"schemaVersion": 1,
		"baselineVersion": 35,
		"migrations": [{
			"version": 36,
			"name": "private_failure",
			"categories": ["backfill"],
			"deployment": "online",
			"backfill": {"mode": "bounded", "resumable": true, "notes": "The fixture operation is finite and retry-safe."},
			"locking": {"risk": "low", "notes": "The fixture table has no concurrent callers."},
			"rollback": {"application": "compatible", "schema": "not_required", "dataLossRisk": false, "notes": "The failed transaction writes no application data."}
		}]
	}`)
	manifest, err := migrationpolicy.LoadDirectory(directory)
	require.NoError(t, err)

	connectionURL, err := url.Parse(dbstore.BuildPostgreSQLDSN(config.Envs))
	require.NoError(t, err)
	query := connectionURL.Query()
	query.Set("search_path", schema)
	connectionURL.RawQuery = query.Encode()
	err = migrateUp(directory, connectionURL.String(), manifest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version=000036 dirty=true")
	assert.NotContains(t, err.Error(), "division by zero")
	assert.NotContains(t, err.Error(), "private_failure")
}

func requireMigrationTestDatabase(t *testing.T, err error) {
	t.Helper()
	if os.Getenv("EXPENSE_TRACKER_REQUIRE_TEST_DB") == "1" {
		t.Fatalf("connect to required migration integration database: %v", err)
	}
	t.Skipf("skipping migration runtime integration: database unavailable: %v", err)
}

func writeMigrationFile(t *testing.T, directory, name, contents string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600))
}

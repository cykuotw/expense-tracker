package migrationpolicy_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandBackfillValidateContractCompatibility(t *testing.T) {
	database, err := dbstore.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		if os.Getenv("EXPENSE_TRACKER_REQUIRE_TEST_DB") == "1" {
			t.Fatalf("connect to required integration database: %v", err)
		}
		t.Skipf("skipping compatibility fixture: db connect error: %v", err)
	}
	defer database.Close()
	if err := database.PingContext(t.Context()); err != nil {
		if os.Getenv("EXPENSE_TRACKER_REQUIRE_TEST_DB") == "1" {
			t.Fatalf("ping required integration database: %v", err)
		}
		t.Skipf("skipping compatibility fixture: db ping error: %v", err)
	}

	tx, err := database.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback()
	schema := strings.ReplaceAll("migration_compatibility_"+uuid.NewString(), "-", "_")
	_, err = tx.ExecContext(
		t.Context(),
		fmt.Sprintf(`CREATE SCHEMA %s; SET LOCAL search_path TO %s`, schema, schema),
	)
	require.NoError(t, err)

	executeFixture(t, tx, "001_initial.sql")
	legacyInsert := `INSERT INTO expense_contract_fixture
		(id, description, total, expense_time) VALUES ($1, $2, $3, $4)`
	firstID := uuid.New()
	_, err = tx.ExecContext(t.Context(), legacyInsert, firstID, "legacy before expansion", "10.000", time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	executeFixture(t, tx, "002_expand.sql")
	secondID := uuid.New()
	_, err = tx.ExecContext(t.Context(), legacyInsert, secondID, "legacy during expansion", "20.000", time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err, "the deployed legacy write contract must survive expansion")
	assertLegacyRead(t, tx, firstID, "legacy before expansion", "10.000")
	assertLegacyRead(t, tx, secondID, "legacy during expansion", "20.000")

	for {
		result, err := tx.ExecContext(t.Context(), `
			WITH batch AS (
				SELECT id
				FROM expense_contract_fixture
				WHERE occurred_on IS NULL
				ORDER BY id
				LIMIT 1
				FOR UPDATE
			)
			UPDATE expense_contract_fixture AS expense
			SET occurred_on = expense.expense_time::date
			FROM batch
			WHERE expense.id = batch.id`)
		require.NoError(t, err)
		updated, err := result.RowsAffected()
		require.NoError(t, err)
		if updated == 0 {
			break
		}
	}

	result, err := tx.ExecContext(t.Context(), `UPDATE expense_contract_fixture
		SET occurred_on = expense_time::date WHERE occurred_on IS NULL`)
	require.NoError(t, err)
	updated, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Zero(t, updated, "a completed backfill must be safe to retry")

	executeFixture(t, tx, "003_constraint.sql")
	thirdID := uuid.New()
	_, err = tx.ExecContext(t.Context(), `INSERT INTO expense_contract_fixture
		(id, description, total, expense_time, occurred_on) VALUES ($1, $2, $3, $4, $5)`,
		thirdID,
		"dual-write application",
		"30.000",
		time.Date(2026, 1, 4, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)

	executeFixture(t, tx, "004_contract.sql")
	assertLegacyRead(t, tx, thirdID, "dual-write application", "30.000")
	var missingOccurrenceDates int
	require.NoError(t, tx.QueryRowContext(
		t.Context(),
		"SELECT COUNT(*) FROM expense_contract_fixture WHERE occurred_on IS NULL",
	).Scan(&missingOccurrenceDates))
	assert.Zero(t, missingOccurrenceDates)
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func executeFixture(t *testing.T, executor sqlExecutor, name string) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("testdata", "compatibility", name))
	require.NoError(t, err)
	_, err = executor.ExecContext(t.Context(), string(contents))
	require.NoError(t, err)
}

func assertLegacyRead(t *testing.T, executor sqlExecutor, id uuid.UUID, description, total string) {
	t.Helper()
	var actualDescription string
	var actualTotal string
	err := executor.QueryRowContext(
		t.Context(),
		"SELECT description, total::text FROM expense_contract_fixture WHERE id = $1",
		id,
	).Scan(&actualDescription, &actualTotal)
	require.NoError(t, err)
	assert.Equal(t, description, actualDescription)
	assert.Equal(t, total, actualTotal)
}

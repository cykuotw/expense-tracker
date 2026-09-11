package main

import (
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

func TestMigration000033BackfillsEveryLegacySplitRule(t *testing.T) {
	database, err := dbstore.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		if os.Getenv("EXPENSE_TRACKER_REQUIRE_TEST_DB") == "1" {
			t.Fatalf("connect to required integration database: %v", err)
		}
		t.Skipf("skipping: db connect error: %v", err)
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		if os.Getenv("EXPENSE_TRACKER_REQUIRE_TEST_DB") == "1" {
			t.Fatalf("ping required integration database: %v", err)
		}
		t.Skipf("skipping: db ping error: %v", err)
	}

	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	up, err := os.ReadFile(filepath.Join(filepath.Dir(sourceFile), "migrations", "000033_add_expense_allocations.up.sql"))
	require.NoError(t, err)

	tx, err := database.Begin()
	require.NoError(t, err)
	defer tx.Rollback()
	schema := strings.ReplaceAll("migration_33_"+uuid.NewString(), "-", "_")
	_, err = tx.Exec(fmt.Sprintf(`CREATE SCHEMA %s; SET LOCAL search_path TO %s`, schema, schema))
	require.NoError(t, err)
	_, err = tx.Exec(`
		CREATE TABLE users (id UUID PRIMARY KEY);
		CREATE TABLE groups (id UUID PRIMARY KEY);
		CREATE TABLE group_member (group_id UUID NOT NULL, user_id UUID NOT NULL);
		CREATE TABLE expense (
			id UUID PRIMARY KEY,
			group_id UUID NOT NULL,
			total NUMERIC(10, 3) NOT NULL,
			split_rule TEXT NOT NULL
		);
		CREATE TABLE ledger (
			id UUID PRIMARY KEY,
			expense_id UUID NOT NULL,
			borrower_user_id UUID NOT NULL,
			share NUMERIC(10, 3) NOT NULL
		);`)
	require.NoError(t, err)

	groupID := uuid.New()
	firstUserID := uuid.New()
	secondUserID := uuid.New()
	_, err = tx.Exec("INSERT INTO users (id) VALUES ($1), ($2)", firstUserID, secondUserID)
	require.NoError(t, err)
	_, err = tx.Exec("INSERT INTO groups (id) VALUES ($1)", groupID)
	require.NoError(t, err)
	_, err = tx.Exec("INSERT INTO group_member (group_id, user_id) VALUES ($1, $2), ($1, $3)", groupID, firstUserID, secondUserID)
	require.NoError(t, err)

	rules := []string{"Equally", "Unequally", "You-Half", "You-Full", "Other-Half", "Other-Full"}
	expenseIDs := make(map[string]uuid.UUID, len(rules))
	for _, rule := range rules {
		expenseID := uuid.New()
		expenseIDs[rule] = expenseID
		_, err = tx.Exec(
			"INSERT INTO expense (id, group_id, total, split_rule) VALUES ($1, $2, 10, $3)",
			expenseID, groupID, rule,
		)
		require.NoError(t, err)
		_, err = tx.Exec(`INSERT INTO ledger (id, expense_id, borrower_user_id, share)
			VALUES ($1, $2, $3, 10), ($4, $2, $5, 0)`,
			uuid.New(), expenseID, firstUserID, uuid.New(), secondUserID)
		require.NoError(t, err)
	}

	_, err = tx.Exec(string(up))
	require.NoError(t, err)

	for _, rule := range rules {
		expectedMode := "exact"
		expectedNonNullAmounts := 2
		if rule == "Equally" || rule == "You-Half" || rule == "Other-Half" {
			expectedMode = "equal"
			expectedNonNullAmounts = 0
		}
		var mode string
		var participantCount int
		var nonNullAmounts int
		var ledgerTotal string
		err = tx.QueryRow(`
			SELECT e.allocation_mode, COUNT(a.user_id), COUNT(a.amount), SUM(l.share)::text
			FROM expense e
			JOIN expense_allocation a ON a.expense_id = e.id
			JOIN ledger l ON l.expense_id = e.id AND l.borrower_user_id = a.user_id
			WHERE e.id = $1
			GROUP BY e.allocation_mode`, expenseIDs[rule]).Scan(
			&mode, &participantCount, &nonNullAmounts, &ledgerTotal,
		)
		require.NoError(t, err)
		assert.Equal(t, expectedMode, mode)
		assert.Equal(t, 2, participantCount)
		assert.Equal(t, expectedNonNullAmounts, nonNullAmounts)
		assert.Equal(t, "10.000", ledgerTotal)
	}

	_, err = tx.Exec("SAVEPOINT duplicate_ledger")
	require.NoError(t, err)
	_, err = tx.Exec(`INSERT INTO ledger (id, expense_id, borrower_user_id, share)
		VALUES ($1, $2, $3, 0)`, uuid.New(), expenseIDs["Equally"], firstUserID)
	require.Error(t, err)
	_, rollbackErr := tx.Exec("ROLLBACK TO SAVEPOINT duplicate_ledger")
	require.NoError(t, rollbackErr)
}

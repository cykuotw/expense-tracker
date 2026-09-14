package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMigration000035IsReversibleAndEnforcesEventShapes(t *testing.T) {
	database := openMigrationTestDB(t)
	up, down := readMigration000035(t)
	tx, err := database.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	schema := strings.ReplaceAll("migration_35_"+uuid.NewString(), "-", "_")
	_, err = tx.Exec(fmt.Sprintf(`
		CREATE SCHEMA %s;
		SET LOCAL search_path TO %s, public;
		CREATE TABLE users (id UUID PRIMARY KEY);
		CREATE TABLE groups (id UUID PRIMARY KEY);
		CREATE TABLE expense (
			id UUID PRIMARY KEY,
			group_id UUID NOT NULL REFERENCES groups(id),
			currency CHAR(3) NOT NULL,
			occurred_on DATE,
			expense_time_utc TIMESTAMPTZ NOT NULL,
			is_deleted BOOLEAN NOT NULL DEFAULT FALSE
		);
		CREATE TABLE web_push_subscription (id UUID PRIMARY KEY);
		CREATE TABLE web_push_delivery (
			id UUID PRIMARY KEY,
			expense_id UUID NOT NULL REFERENCES expense(id) ON DELETE CASCADE,
			subscription_id UUID NOT NULL REFERENCES web_push_subscription(id) ON DELETE CASCADE,
			recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
			actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			group_name VARCHAR(32) NOT NULL,
			currency CHAR(3) NOT NULL,
			amount NUMERIC(10, 3) NOT NULL,
			attempts SMALLINT NOT NULL DEFAULT 0,
			available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMPTZ NOT NULL,
			delivered_at TIMESTAMPTZ,
			completed_at TIMESTAMPTZ,
			failure_code VARCHAR(32),
			claim_token UUID,
			claimed_until TIMESTAMPTZ,
			UNIQUE (expense_id, subscription_id)
		);`, schema, schema))
	require.NoError(t, err)
	require.NoError(t, executeSQLScript(tx, string(up)))
	var expensePageIndexExists bool
	require.NoError(t, tx.QueryRow("SELECT to_regclass(current_schema() || '.expense_monthly_review_page_idx') IS NOT NULL").Scan(&expensePageIndexExists))
	require.True(t, expensePageIndexExists)

	userID, groupID, subscriptionID := uuid.New(), uuid.New(), uuid.New()
	for query, id := range map[string]uuid.UUID{
		"INSERT INTO users (id) VALUES ($1)":                 userID,
		"INSERT INTO groups (id) VALUES ($1)":                groupID,
		"INSERT INTO web_push_subscription (id) VALUES ($1)": subscriptionID,
	} {
		_, err = tx.Exec(query, id)
		require.NoError(t, err)
	}
	month := "2026-08-01"
	_, err = tx.Exec("INSERT INTO monthly_review_publication (group_id, review_month) VALUES ($1, $2)", groupID, month)
	require.NoError(t, err)
	_, err = tx.Exec(`INSERT INTO web_push_delivery (
		id, notification_type, review_month, subscription_id, recipient_user_id,
		group_id, group_name, expires_at
	) VALUES ($1, 'monthly_review_available', $2, $3, $4, $5, 'Home', NOW() + INTERVAL '1 day')`,
		uuid.New(), month, subscriptionID, userID, groupID)
	require.NoError(t, err)

	_, err = tx.Exec("SAVEPOINT invalid_event")
	require.NoError(t, err)
	_, err = tx.Exec(`INSERT INTO web_push_delivery (
		id, notification_type, subscription_id, recipient_user_id, group_id, group_name, expires_at
	) VALUES ($1, 'monthly_review_available', $2, $3, $4, 'Home', NOW() + INTERVAL '1 day')`,
		uuid.New(), subscriptionID, userID, groupID)
	require.Error(t, err)
	_, rollbackErr := tx.Exec("ROLLBACK TO SAVEPOINT invalid_event")
	require.NoError(t, rollbackErr)

	require.NoError(t, executeSQLScript(tx, string(down)))
	var tableExists bool
	require.NoError(t, tx.QueryRow("SELECT to_regclass(current_schema() || '.monthly_review_publication') IS NOT NULL").Scan(&tableExists))
	require.False(t, tableExists)
	var notificationColumnExists bool
	require.NoError(t, tx.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'web_push_delivery' AND column_name = 'notification_type'
	)`).Scan(&notificationColumnExists))
	require.False(t, notificationColumnExists)
	require.NoError(t, tx.QueryRow("SELECT to_regclass(current_schema() || '.expense_monthly_review_page_idx') IS NOT NULL").Scan(&expensePageIndexExists))
	require.False(t, expensePageIndexExists)
}

func readMigration000035(t *testing.T) ([]byte, []byte) {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrationDir := filepath.Join(filepath.Dir(sourceFile), "migrations")
	up, err := os.ReadFile(filepath.Join(migrationDir, "000035_add_monthly_expense_reviews.up.sql"))
	require.NoError(t, err)
	down, err := os.ReadFile(filepath.Join(migrationDir, "000035_add_monthly_expense_reviews.down.sql"))
	require.NoError(t, err)
	return up, down
}

func executeSQLScript(tx *sql.Tx, script string) error {
	_, err := tx.Exec(script)
	return err
}

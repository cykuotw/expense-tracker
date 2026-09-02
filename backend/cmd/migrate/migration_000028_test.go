package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration000028TemporalContract(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrationDir := filepath.Join(filepath.Dir(sourceFile), "migrations")

	up, err := os.ReadFile(filepath.Join(migrationDir, "000028_correct_expense_date_and_timestamp_semantics.up.sql"))
	require.NoError(t, err)
	down, err := os.ReadFile(filepath.Join(migrationDir, "000028_correct_expense_date_and_timestamp_semantics.down.sql"))
	require.NoError(t, err)

	upSQL := strings.ToLower(string(up))
	downSQL := strings.ToLower(string(down))
	assert.Contains(t, upSQL, "add column occurred_on date")
	assert.Contains(t, upSQL, "type timestamp with time zone")
	assert.Contains(t, upSQL, "at time zone 'utc'")
	assert.NotContains(t, upSQL, "alter table users")
	assert.Contains(t, downSQL, "drop column occurred_on")
	assert.Contains(t, downSQL, "type timestamp without time zone")
}

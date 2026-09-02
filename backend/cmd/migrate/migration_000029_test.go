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

func TestMigration000029RefreshTokenFamilies(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrationDir := filepath.Join(filepath.Dir(sourceFile), "migrations")

	up, err := os.ReadFile(filepath.Join(migrationDir, "000029_make_refresh_token_rotation_atomic.up.sql"))
	require.NoError(t, err)
	down, err := os.ReadFile(filepath.Join(migrationDir, "000029_make_refresh_token_rotation_atomic.down.sql"))
	require.NoError(t, err)

	upSQL := strings.ToLower(string(up))
	downSQL := strings.ToLower(string(down))
	addPosition := strings.Index(upSQL, "add column family_id uuid")
	backfillPosition := strings.Index(upSQL, "set family_id = id")
	notNullPosition := strings.Index(upSQL, "alter column family_id set not null")

	assert.GreaterOrEqual(t, addPosition, 0)
	assert.Greater(t, backfillPosition, addPosition)
	assert.Greater(t, notNullPosition, backfillPosition)
	assert.Contains(t, upSQL, "create index idx_refresh_tokens_family_id")
	assert.Contains(t, downSQL, "drop index idx_refresh_tokens_family_id")
	assert.Contains(t, downSQL, "drop column family_id")
}

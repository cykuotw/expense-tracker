package migrationpolicy

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryManifestMatchesCurrentMigrations(t *testing.T) {
	directory := filepath.Join("..", "..", "cmd", "migrate", "migrations")
	manifest, err := LoadDirectory(directory)
	require.NoError(t, err)
	assert.Equal(t, BaselineVersion, manifest.BaselineVersion)
	assert.Empty(t, manifest.Migrations)
}

func TestSharedValidFixture(t *testing.T) {
	manifest, err := LoadDirectory(filepath.Join("testdata", "valid"))
	require.NoError(t, err)
	require.Len(t, manifest.Migrations, 1)
	assert.NoError(t, manifest.ValidatePending(BaselineVersion, false))
}

func TestSharedDestructiveFixtureIsRejected(t *testing.T) {
	_, err := LoadDirectory(filepath.Join("testdata", "invalid_destructive"))
	require.ErrorContains(t, err, "requires one of categories drop")
}

func TestDirtyAndMaintenancePendingMigrationsAreRejected(t *testing.T) {
	manifest, err := LoadDirectory(filepath.Join("testdata", "valid"))
	require.NoError(t, err)
	require.ErrorContains(t, manifest.ValidatePending(35, true), "version=000035 dirty=true")

	manifest.Migrations[0].Deployment = "maintenance_required"
	require.ErrorContains(t, manifest.ValidatePending(35, false), "maintenance-required")
	assert.NoError(t, manifest.ValidatePending(36, false))
}

func TestMissingManifestFieldIsRejected(t *testing.T) {
	directory := t.TempDir()
	writeMigrationPair(t, directory, 35, "baseline", "SELECT 1;")
	require.NoError(t, os.WriteFile(
		filepath.Join(directory, ManifestName),
		[]byte(`{"schemaVersion":1,"baselineVersion":35,"migrations":null}`),
		0o600,
	))

	_, err := LoadDirectory(directory)
	require.ErrorContains(t, err, "migrations must be an array")
}

func TestDataRewriteRequiresBoundedBackfill(t *testing.T) {
	resumable := false
	noDataLoss := false
	entry := MigrationEntry{
		Version:    36,
		Name:       "rewrite",
		Categories: []string{"data_rewrite"},
		Deployment: "online",
		Backfill: BackfillPolicy{
			Mode:      "none",
			Resumable: &resumable,
			Notes:     "Existing rows would be rewritten.",
		},
		Locking: LockingPolicy{Risk: "medium", Notes: "Bounded lock assessment."},
		Rollback: RollbackPolicy{
			Application:  "compatible",
			Schema:       "manual_only",
			DataLossRisk: &noDataLoss,
			Notes:        "Application rollback remains compatible.",
		},
	}

	err := validateEntry(entry, 0)
	require.ErrorContains(t, err, "requires bounded mode")
}

func TestDestructiveSQLRequiresMatchingCategory(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		category string
	}{
		{name: "truncate", sql: "TRUNCATE TABLE expense;", category: "data_rewrite"},
		{name: "drop view", sql: "DROP MATERIALIZED VIEW expense_summary;", category: "drop"},
		{name: "rename table", sql: "ALTER TABLE expense RENAME TO expense_legacy;", category: "rename"},
		{name: "rename column without keyword", sql: "ALTER TABLE expense RENAME note TO description;", category: "rename"},
		{name: "rename index", sql: "ALTER INDEX expense_idx RENAME TO expense_legacy_idx;", category: "rename"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entry := MigrationEntry{Version: 36, Name: "unsafe", Categories: []string{"additive"}}

			err := validateSQLConsistency(entry, test.sql)

			require.ErrorContains(t, err, test.category)
		})
	}
}

func writeMigrationPair(t *testing.T, directory string, version int, name, up string) {
	t.Helper()
	prefix := filepath.Join(directory, formatVersion(version)+"_"+name)
	require.NoError(t, os.WriteFile(prefix+".up.sql", []byte(up), 0o600))
	require.NoError(t, os.WriteFile(prefix+".down.sql", []byte("SELECT 1;"), 0o600))
}

func formatVersion(version int) string {
	return fmt.Sprintf("%06d", version)
}

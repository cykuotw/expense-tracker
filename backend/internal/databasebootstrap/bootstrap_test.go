package databasebootstrap

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"expense-tracker/backend/services/user"

	"github.com/golang-migrate/migrate/v4"
	migratedatabase "github.com/golang-migrate/migrate/v4/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig() Config {
	return Config{
		Host:              "db.internal",
		Port:              "5432",
		SSLMode:           "disable",
		MaintenanceName:   "postgres",
		DatabaseName:      "expense_tracker",
		AdminUser:         "postgres",
		AdminPassword:     "admin-secret",
		MigrationUser:     "expense_migration",
		MigrationPassword: "migration-secret",
		RuntimeUser:       "expense_runtime",
		RuntimePassword:   "runtime-secret",
		MigrationsPath:    "/var/task/migrations",
	}
}

func TestConfigValidate(t *testing.T) {
	assert.NoError(t, validConfig().Validate())
}

func TestConfigValidateRejectsInvalidDatabaseSettings(t *testing.T) {
	cfg := validConfig()
	cfg.RuntimePassword = ""
	assert.EqualError(t, cfg.Validate(), "DB_RUNTIME_PASSWORD is required")

	cfg = validConfig()
	cfg.RuntimeUser = cfg.MigrationUser
	assert.EqualError(t, cfg.Validate(), "database admin, migration, and runtime users must be distinct")

	cfg = validConfig()
	cfg.DatabaseName = cfg.MaintenanceName
	assert.EqualError(t, cfg.Validate(), "DB_NAME must differ from DB_MAINTENANCE_NAME")
}

func TestConfigAllowsOptionalFirstAdmin(t *testing.T) {
	cfg := validConfig()
	cfg.FirstAdmin = &user.FirstAdminInput{
		Email:     "admin@example.com",
		Password:  "long-enough-password",
		Firstname: "Admin",
		Lastname:  "User",
	}
	assert.NoError(t, cfg.Validate())
}

func TestRunRejectsInvalidMigrationPolicyBeforeDatabasePreparation(t *testing.T) {
	cfg := validConfig()
	cfg.MigrationsPath = t.TempDir()

	_, err := Run(t.Context(), cfg)

	assert.ErrorContains(t, err, "validate migration policy")
	assert.ErrorContains(t, err, "manifest.json")
}

func TestRunInspectsDatabaseStateBeforePreparingMaintenanceMigration(t *testing.T) {
	cfg := validConfig()
	cfg.Host = "127.0.0.1"
	cfg.Port = "1"
	cfg.MigrationsPath = filepath.Join("..", "migrationpolicy", "testdata", "valid")
	contents, err := os.ReadFile(filepath.Join(cfg.MigrationsPath, "manifest.json"))
	require.NoError(t, err)
	maintenanceManifest := strings.Replace(
		string(contents),
		`"deployment": "online"`,
		`"deployment": "maintenance_required"`,
		1,
	)
	directory := t.TempDir()
	entries, err := os.ReadDir(cfg.MigrationsPath)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.Name() == "manifest.json" {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(cfg.MigrationsPath, entry.Name()))
		require.NoError(t, readErr)
		require.NoError(t, os.WriteFile(filepath.Join(directory, entry.Name()), data, 0o600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(maintenanceManifest), 0o600))
	cfg.MigrationsPath = directory

	_, err = Run(t.Context(), cfg)

	assert.ErrorContains(t, err, "connect to migration database for state inspection failed")
}

func TestSQLQuoting(t *testing.T) {
	assert.Equal(t, `"role""name"`, quoteIdentifier(`role"name`))
	assert.Equal(t, `'pa''ss'`, quoteLiteral(`pa'ss`))
}

func TestMigrationConnectionStringAppliesBoundedTimeouts(t *testing.T) {
	connectionString, err := migrationConnectionString(
		"postgres://migration:secret@db.internal:5432/expense_tracker?sslmode=require",
	)
	assert.NoError(t, err)
	parsed, err := url.Parse(connectionString)
	assert.NoError(t, err)
	assert.Equal(t, "require", parsed.Query().Get("sslmode"))
	assert.Equal(t, "-c lock_timeout=5s -c statement_timeout=240s", parsed.Query().Get("options"))
}

func TestMigrationConnectionStringDoesNotExposeInvalidInput(t *testing.T) {
	_, err := migrationConnectionString("postgres://migration:private-secret@%zz")
	if assert.Error(t, err) {
		assert.False(t, strings.Contains(err.Error(), "private-secret"))
	}
}

type migrationStateStub struct {
	version uint
	dirty   bool
	err     error
}

func (stub migrationStateStub) Version() (uint, bool, error) {
	return stub.version, stub.dirty, stub.err
}

type sqlStateStub struct {
	code   string
	detail string
}

func (stub sqlStateStub) Error() string {
	return stub.detail
}

func (stub sqlStateStub) SQLState() string {
	return stub.code
}

func TestMigrationExecutionErrorReportsOnlyRecoveryState(t *testing.T) {
	err := migrationExecutionError(
		errors.New("private database detail"),
		migrationStateStub{version: 36, dirty: true},
	)
	assert.EqualError(
		t,
		err,
		"migration execution failed: reason=sql_error version=000036 dirty=true; inspect migration recovery guidance",
	)

	err = migrationExecutionError(
		errors.New("private database detail"),
		migrationStateStub{err: errors.New("private database detail")},
	)
	assert.EqualError(
		t,
		err,
		"migration execution failed: reason=sql_error; database migration state could not be read",
	)
	assert.NotContains(t, err.Error(), "private database detail")
}

func TestMigrationExecutionErrorClassifiesTimeoutsWithoutDetails(t *testing.T) {
	tests := []struct {
		name   string
		cause  error
		reason string
	}{
		{name: "migration lock", cause: migrate.ErrLockTimeout, reason: "migration_lock_timeout"},
		{name: "postgres lock", cause: sqlStateStub{code: "55P03", detail: "private lock detail"}, reason: "lock_timeout"},
		{name: "statement", cause: sqlStateStub{code: "57014", detail: "private query detail"}, reason: "statement_timeout"},
		{
			name: "wrapped statement",
			cause: migratedatabase.Error{
				OrigErr: sqlStateStub{code: "57014", detail: "private query detail"},
				Query:   []byte("private SQL"),
			},
			reason: "statement_timeout",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := migrationExecutionError(test.cause, migrationStateStub{version: 36, dirty: true})

			assert.ErrorContains(t, err, "reason="+test.reason)
			assert.NotContains(t, err.Error(), "private")
		})
	}
}

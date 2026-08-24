package databasebootstrap

import (
	"testing"

	"expense-tracker/backend/services/user"

	"github.com/stretchr/testify/assert"
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

func TestSQLQuoting(t *testing.T) {
	assert.Equal(t, `"role""name"`, quoteIdentifier(`role"name`))
	assert.Equal(t, `'pa''ss'`, quoteLiteral(`pa'ss`))
}

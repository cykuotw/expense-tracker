package main

import (
	"context"
	"errors"
	"testing"

	"expense-tracker/backend/internal/databasebootstrap"

	"github.com/stretchr/testify/assert"
)

func TestHandlerDefaultsEmptyOperationToAll(t *testing.T) {
	called := false
	handler := newHandler(
		func() (databasebootstrap.Config, error) { return databasebootstrap.Config{}, nil },
		func(context.Context, databasebootstrap.Config) (databasebootstrap.Result, error) {
			called = true
			return databasebootstrap.Result{FirstAdminStatus: "not_requested"}, nil
		},
	)

	result, err := handler(context.Background(), request{})

	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, response{Status: "ok", Operation: "all", FirstAdminStatus: "not_requested"}, result)
}

func TestHandlerRejectsUnknownOperationBeforeLoadingConfig(t *testing.T) {
	handler := newHandler(
		func() (databasebootstrap.Config, error) {
			t.Fatal("config loader should not run")
			return databasebootstrap.Config{}, nil
		},
		nil,
	)

	_, err := handler(context.Background(), request{Operation: "migrate"})

	assert.EqualError(t, err, `unsupported operation "migrate"`)
}

func TestHandlerReturnsRunnerError(t *testing.T) {
	expected := errors.New("database unavailable")
	handler := newHandler(
		func() (databasebootstrap.Config, error) { return databasebootstrap.Config{}, nil },
		func(context.Context, databasebootstrap.Config) (databasebootstrap.Result, error) {
			return databasebootstrap.Result{}, expected
		},
	)

	_, err := handler(context.Background(), request{Operation: "all"})

	assert.ErrorIs(t, err, expected)
}

func TestLoadConfigSupportsOptionalFirstAdmin(t *testing.T) {
	setRequiredDatabaseEnvironment(t)
	t.Setenv("LAMBDA_TASK_ROOT", "/var/task")

	cfg, err := loadConfig()
	assert.NoError(t, err)
	assert.Nil(t, cfg.FirstAdmin)

	t.Setenv("FIRST_ADMIN_EMAIL", "admin@example.com")
	t.Setenv("FIRST_ADMIN_PASSWORD", "long-enough-password")
	t.Setenv("FIRST_ADMIN_FIRSTNAME", "Admin")
	t.Setenv("FIRST_ADMIN_LASTNAME", "User")
	t.Setenv("FIRST_ADMIN_NICKNAME", "admin")

	cfg, err = loadConfig()
	assert.NoError(t, err)
	assert.Equal(t, "admin@example.com", cfg.FirstAdmin.Email)
	assert.Equal(t, "/var/task/migrations", cfg.MigrationsPath)
}

func TestLoadConfigRejectsPartialFirstAdmin(t *testing.T) {
	setRequiredDatabaseEnvironment(t)
	t.Setenv("LAMBDA_TASK_ROOT", "/var/task")
	t.Setenv("FIRST_ADMIN_EMAIL", "admin@example.com")

	_, err := loadConfig()

	assert.EqualError(t, err, "FIRST_ADMIN_PASSWORD is required")
}

func setRequiredDatabaseEnvironment(t *testing.T) {
	t.Helper()
	values := map[string]string{
		"DB_PUBLIC_HOST":        "10.0.0.10",
		"DB_PORT":               "5432",
		"DB_SSLMODE":            "disable",
		"DB_MAINTENANCE_NAME":   "postgres",
		"DB_NAME":               "expense_tracker",
		"DB_ADMIN_USER":         "postgres",
		"DB_ADMIN_PASSWORD":     "admin-secret",
		"DB_MIGRATION_USER":     "expense_migration",
		"DB_MIGRATION_PASSWORD": "migration-secret",
		"DB_RUNTIME_USER":       "expense_runtime",
		"DB_RUNTIME_PASSWORD":   "runtime-secret",
	}
	for name, value := range values {
		t.Setenv(name, value)
	}
}

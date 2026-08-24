package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"expense-tracker/backend/internal/databasebootstrap"
	"expense-tracker/backend/services/user"

	"github.com/aws/aws-lambda-go/lambda"
)

type request struct {
	Operation string `json:"operation"`
}

type response struct {
	Status           string `json:"status"`
	Operation        string `json:"operation"`
	FirstAdminStatus string `json:"first_admin_status"`
}

type configLoader func() (databasebootstrap.Config, error)
type bootstrapRunner func(context.Context, databasebootstrap.Config) (databasebootstrap.Result, error)

func main() {
	lambda.Start(newHandler(loadConfig, databasebootstrap.Run))
}

func newHandler(load configLoader, run bootstrapRunner) func(context.Context, request) (response, error) {
	return func(ctx context.Context, input request) (response, error) {
		operation := strings.TrimSpace(input.Operation)
		if operation == "" {
			operation = "all"
		}
		if operation != "all" {
			return response{}, fmt.Errorf("unsupported operation %q", operation)
		}

		cfg, err := load()
		if err != nil {
			return response{}, err
		}
		result, err := run(ctx, cfg)
		if err != nil {
			return response{}, fmt.Errorf("bootstrap failed: %w", err)
		}
		return response{
			Status:           "ok",
			Operation:        operation,
			FirstAdminStatus: result.FirstAdminStatus,
		}, nil
	}
}

func loadConfig() (databasebootstrap.Config, error) {
	required := func(name string) (string, error) {
		value := strings.TrimSpace(os.Getenv(name))
		if value == "" {
			return "", fmt.Errorf("%s is required", name)
		}
		return value, nil
	}

	values := make(map[string]string)
	for _, name := range []string{
		"DB_PUBLIC_HOST",
		"DB_PORT",
		"DB_SSLMODE",
		"DB_MAINTENANCE_NAME",
		"DB_NAME",
		"DB_ADMIN_USER",
		"DB_ADMIN_PASSWORD",
		"DB_MIGRATION_USER",
		"DB_MIGRATION_PASSWORD",
		"DB_RUNTIME_USER",
		"DB_RUNTIME_PASSWORD",
	} {
		value, err := required(name)
		if err != nil {
			return databasebootstrap.Config{}, err
		}
		values[name] = value
	}

	firstAdmin, err := loadFirstAdmin(required)
	if err != nil {
		return databasebootstrap.Config{}, err
	}
	root := strings.TrimSpace(os.Getenv("LAMBDA_TASK_ROOT"))
	if root == "" {
		executable, err := os.Executable()
		if err != nil {
			return databasebootstrap.Config{}, fmt.Errorf("locate executable: %w", err)
		}
		root = filepath.Dir(executable)
	}

	cfg := databasebootstrap.Config{
		Host:              values["DB_PUBLIC_HOST"],
		Port:              values["DB_PORT"],
		SSLMode:           values["DB_SSLMODE"],
		MaintenanceName:   values["DB_MAINTENANCE_NAME"],
		DatabaseName:      values["DB_NAME"],
		AdminUser:         values["DB_ADMIN_USER"],
		AdminPassword:     values["DB_ADMIN_PASSWORD"],
		MigrationUser:     values["DB_MIGRATION_USER"],
		MigrationPassword: values["DB_MIGRATION_PASSWORD"],
		RuntimeUser:       values["DB_RUNTIME_USER"],
		RuntimePassword:   values["DB_RUNTIME_PASSWORD"],
		MigrationsPath:    filepath.Join(root, "migrations"),
		FirstAdmin:        firstAdmin,
	}
	if err := cfg.Validate(); err != nil {
		return databasebootstrap.Config{}, err
	}
	return cfg, nil
}

func loadFirstAdmin(required func(string) (string, error)) (*user.FirstAdminInput, error) {
	names := []string{
		"FIRST_ADMIN_EMAIL",
		"FIRST_ADMIN_PASSWORD",
		"FIRST_ADMIN_FIRSTNAME",
		"FIRST_ADMIN_LASTNAME",
		"FIRST_ADMIN_NICKNAME",
	}
	requested := false
	for _, name := range names {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			requested = true
			break
		}
	}
	if !requested {
		return nil, nil
	}

	values := make(map[string]string)
	for _, name := range names[:4] {
		value, err := required(name)
		if err != nil {
			return nil, err
		}
		values[name] = value
	}
	return &user.FirstAdminInput{
		Email:     values["FIRST_ADMIN_EMAIL"],
		Password:  values["FIRST_ADMIN_PASSWORD"],
		Firstname: values["FIRST_ADMIN_FIRSTNAME"],
		Lastname:  values["FIRST_ADMIN_LASTNAME"],
		Nickname:  strings.TrimSpace(os.Getenv("FIRST_ADMIN_NICKNAME")),
	}, nil
}

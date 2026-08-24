package main

import (
	"context"
	"log"
	"os"
	"strings"

	"expense-tracker/backend/internal/databasebootstrap"
)

func main() {
	cfg := databasebootstrap.Config{
		Host:              requireEnv("DB_PUBLIC_HOST"),
		Port:              requireEnv("DB_PORT"),
		SSLMode:           requireEnv("DB_SSLMODE"),
		MaintenanceName:   envOrDefault("DB_MAINTENANCE_NAME", "postgres"),
		DatabaseName:      requireEnv("DB_NAME"),
		AdminUser:         requireEnv("DB_ADMIN_USER"),
		AdminPassword:     requireEnv("DB_ADMIN_PASSWORD"),
		MigrationUser:     requireEnv("DB_MIGRATION_USER"),
		MigrationPassword: requireEnv("DB_MIGRATION_PASSWORD"),
		RuntimeUser:       requireEnv("DB_APP_USER"),
		RuntimePassword:   requireEnv("DB_APP_PASSWORD"),
	}
	if err := databasebootstrap.Prepare(context.Background(), cfg); err != nil {
		log.Fatal(err)
	}
	log.Println("database roles and grants prepared")
}

func requireEnv(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		log.Fatalf("missing required environment variable %s", name)
	}
	return value
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

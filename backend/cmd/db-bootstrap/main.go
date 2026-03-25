package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type bootstrapConfig struct {
	host              string
	port              string
	dbName            string
	sslMode           string
	adminUser         string
	adminPassword     string
	migrationUser     string
	migrationPassword string
	appUser           string
	appPassword       string
}

func main() {
	cfg := loadConfig()

	adminDB, err := openDB(dsnFor(cfg.adminUser, cfg.adminPassword, cfg))
	if err != nil {
		log.Fatal(err)
	}
	defer adminDB.Close()

	if err := bootstrapAsAdmin(adminDB, cfg); err != nil {
		log.Fatal(err)
	}

	migrationDB, err := openDB(dsnFor(cfg.migrationUser, cfg.migrationPassword, cfg))
	if err != nil {
		log.Fatal(err)
	}
	defer migrationDB.Close()

	if err := applyDefaultPrivileges(migrationDB, cfg); err != nil {
		log.Fatal(err)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func dsnFor(username string, password string, cfg bootstrapConfig) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		username,
		password,
		cfg.host,
		cfg.port,
		cfg.dbName,
		cfg.sslMode,
	)
}

func loadConfig() bootstrapConfig {
	requireEnv := func(key string) string {
		value, ok := os.LookupEnv(key)
		if !ok || strings.TrimSpace(value) == "" {
			log.Fatalf("missing required environment variable %s", key)
		}
		return value
	}

	cfg := bootstrapConfig{
		host:              requireEnv("DB_PUBLIC_HOST"),
		port:              requireEnv("DB_PORT"),
		dbName:            requireEnv("DB_NAME"),
		sslMode:           requireEnv("DB_SSLMODE"),
		adminUser:         requireEnv("DB_ADMIN_USER"),
		adminPassword:     requireEnv("DB_ADMIN_PASSWORD"),
		migrationUser:     requireEnv("DB_MIGRATION_USER"),
		migrationPassword: requireEnv("DB_MIGRATION_PASSWORD"),
		appUser:           requireEnv("DB_APP_USER"),
		appPassword:       requireEnv("DB_APP_PASSWORD"),
	}

	if cfg.adminUser == cfg.migrationUser || cfg.adminUser == cfg.appUser || cfg.migrationUser == cfg.appUser {
		log.Fatal("DB admin, migration, and app users must be distinct")
	}

	return cfg
}

func bootstrapAsAdmin(db *sql.DB, cfg bootstrapConfig) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statements := make([]string, 0, 11)

	var migrationExists bool
	if err := tx.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", cfg.migrationUser).Scan(&migrationExists); err != nil {
		return err
	}
	if !migrationExists {
		statements = append(statements, fmt.Sprintf(
			"CREATE ROLE %s WITH LOGIN PASSWORD %s NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT",
			quoteSQL(cfg.migrationUser, false),
			quoteSQL(cfg.migrationPassword, true),
		))
	}

	var appExists bool
	if err := tx.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", cfg.appUser).Scan(&appExists); err != nil {
		return err
	}
	if !appExists {
		statements = append(statements, fmt.Sprintf(
			"CREATE ROLE %s WITH LOGIN PASSWORD %s NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT",
			quoteSQL(cfg.appUser, false),
			quoteSQL(cfg.appPassword, true),
		))
	}

	statements = append(statements,
		"CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public",
		fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", quoteSQL(cfg.dbName, false), quoteSQL(cfg.migrationUser, false)),
		fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", quoteSQL(cfg.dbName, false), quoteSQL(cfg.appUser, false)),
		fmt.Sprintf("GRANT USAGE, CREATE ON SCHEMA public TO %s", quoteSQL(cfg.migrationUser, false)),
		fmt.Sprintf("ALTER SCHEMA public OWNER TO %s", quoteSQL(cfg.migrationUser, false)),
		fmt.Sprintf("GRANT USAGE ON SCHEMA public TO %s", quoteSQL(cfg.appUser, false)),
		fmt.Sprintf("GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %s", quoteSQL(cfg.appUser, false)),
		fmt.Sprintf("GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO %s", quoteSQL(cfg.appUser, false)),
		fmt.Sprintf("GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO %s", quoteSQL(cfg.appUser, false)),
	)

	for _, statement := range statements {
		log.Printf("running as admin: %s", statement)
		if _, err := tx.Exec(statement); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func applyDefaultPrivileges(db *sql.DB, cfg bootstrapConfig) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statements := []string{
		fmt.Sprintf(
			"ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %s",
			quoteSQL(cfg.migrationUser, false),
			quoteSQL(cfg.appUser, false),
		),
		fmt.Sprintf(
			"ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA public GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO %s",
			quoteSQL(cfg.migrationUser, false),
			quoteSQL(cfg.appUser, false),
		),
		fmt.Sprintf(
			"ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO %s",
			quoteSQL(cfg.migrationUser, false),
			quoteSQL(cfg.appUser, false),
		),
	}

	for _, statement := range statements {
		log.Printf("running as migration user: %s", statement)
		if _, err := tx.Exec(statement); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func quoteSQL(value string, literal bool) string {
	quote := `"`
	escapedQuote := `""`
	if literal {
		quote = `'`
		escapedQuote = `''`
	}
	return quote + strings.ReplaceAll(value, quote, escapedQuote) + quote
}

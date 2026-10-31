package databasebootstrap

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

const versionTable = "expense_agent_schema_migrations"

func migrateUp(parent context.Context, db *sql.DB, files fs.FS) error {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SET statement_timeout = '30s'; SET lock_timeout = '10s'`); err != nil {
		return err
	}
	driver, err := postgres.WithConnection(ctx, conn, &postgres.Config{
		SchemaName:       "public",
		MigrationsTable:  versionTable,
		StatementTimeout: 30 * time.Second,
	})
	if err != nil {
		return err
	}
	source, err := iofs.New(files, "migrations")
	if err != nil {
		driver.Close()
		return err
	}
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		source.Close()
		driver.Close()
		return err
	}
	defer m.Close()
	m.LockTimeout = 10 * time.Second
	_, _, err = m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		var exists bool
		if err := conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='expense_agent')`).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return errors.New("refusing to adopt an untracked existing expense_agent schema")
		}
	} else if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

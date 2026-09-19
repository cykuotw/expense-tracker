package databasebootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"

	"expense-tracker/backend/internal/migrationpolicy"
	"expense-tracker/backend/services/user"

	"github.com/golang-migrate/migrate/v4"
	migratedatabase "github.com/golang-migrate/migrate/v4/database"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	Host              string
	Port              string
	SSLMode           string
	MaintenanceName   string
	DatabaseName      string
	AdminUser         string
	AdminPassword     string
	MigrationUser     string
	MigrationPassword string
	RuntimeUser       string
	RuntimePassword   string
	MigrationsPath    string
	FirstAdmin        *user.FirstAdminInput
}

type Result struct {
	FirstAdminStatus string
}

type MigrationState struct {
	Version uint
	Dirty   bool
}

func (cfg Config) Validate() error {
	if err := cfg.validateDatabaseSettings(); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.MigrationsPath) == "" {
		return errors.New("migrations path is required")
	}
	return nil
}

func (cfg Config) validateDatabaseSettings() error {
	required := map[string]string{
		"DB_PUBLIC_HOST":        cfg.Host,
		"DB_PORT":               cfg.Port,
		"DB_SSLMODE":            cfg.SSLMode,
		"DB_MAINTENANCE_NAME":   cfg.MaintenanceName,
		"DB_NAME":               cfg.DatabaseName,
		"DB_ADMIN_USER":         cfg.AdminUser,
		"DB_ADMIN_PASSWORD":     cfg.AdminPassword,
		"DB_MIGRATION_USER":     cfg.MigrationUser,
		"DB_MIGRATION_PASSWORD": cfg.MigrationPassword,
		"DB_RUNTIME_USER":       cfg.RuntimeUser,
		"DB_RUNTIME_PASSWORD":   cfg.RuntimePassword,
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if cfg.DatabaseName == cfg.MaintenanceName {
		return errors.New("DB_NAME must differ from DB_MAINTENANCE_NAME")
	}
	if cfg.AdminUser == cfg.MigrationUser || cfg.AdminUser == cfg.RuntimeUser || cfg.MigrationUser == cfg.RuntimeUser {
		return errors.New("database admin, migration, and runtime users must be distinct")
	}
	return nil
}

func Prepare(ctx context.Context, cfg Config) error {
	if err := cfg.validateDatabaseSettings(); err != nil {
		return err
	}

	maintenanceDB, err := open(ctx, dsn(cfg.AdminUser, cfg.AdminPassword, cfg, cfg.MaintenanceName))
	if err != nil {
		return fmt.Errorf("connect to maintenance database: %w", err)
	}
	if err := ensureDatabase(ctx, maintenanceDB, cfg.DatabaseName); err != nil {
		maintenanceDB.Close()
		return fmt.Errorf("ensure application database: %w", err)
	}
	if err := maintenanceDB.Close(); err != nil {
		return fmt.Errorf("close maintenance database: %w", err)
	}

	adminDB, err := open(ctx, dsn(cfg.AdminUser, cfg.AdminPassword, cfg, cfg.DatabaseName))
	if err != nil {
		return fmt.Errorf("connect to application database as admin: %w", err)
	}
	defer adminDB.Close()
	if err := reconcileRolesAndBaseGrants(ctx, adminDB, cfg); err != nil {
		return fmt.Errorf("reconcile roles and grants: %w", err)
	}

	migrationDB, err := open(ctx, dsn(cfg.MigrationUser, cfg.MigrationPassword, cfg, cfg.DatabaseName))
	if err != nil {
		return fmt.Errorf("validate migration login: %w", err)
	}
	if err := applyDefaultPrivileges(ctx, migrationDB, cfg); err != nil {
		migrationDB.Close()
		return fmt.Errorf("apply default privileges: %w", err)
	}
	if err := migrationDB.Close(); err != nil {
		return fmt.Errorf("close migration database: %w", err)
	}
	return nil
}

func Run(ctx context.Context, cfg Config) (Result, error) {
	if err := cfg.Validate(); err != nil {
		return Result{}, err
	}
	manifest, err := migrationpolicy.LoadDirectory(cfg.MigrationsPath)
	if err != nil {
		return Result{}, fmt.Errorf("validate migration policy: %w", err)
	}
	if manifest.HasMaintenanceRequired() {
		state, stateErr := ReadMigrationState(ctx, cfg)
		if stateErr != nil {
			return Result{}, stateErr
		}
		if err := manifest.ValidatePending(state.Version, state.Dirty); err != nil {
			return Result{}, fmt.Errorf("validate migration policy: %w", err)
		}
	}
	if err := Prepare(ctx, cfg); err != nil {
		return Result{}, err
	}

	migrationDSN := dsn(cfg.MigrationUser, cfg.MigrationPassword, cfg, cfg.DatabaseName)
	if err := migrateUp(cfg.MigrationsPath, migrationDSN, manifest); err != nil {
		return Result{}, fmt.Errorf("apply migrations: %w", err)
	}

	adminDB, err := open(ctx, dsn(cfg.AdminUser, cfg.AdminPassword, cfg, cfg.DatabaseName))
	if err != nil {
		return Result{}, fmt.Errorf("reconnect to application database as admin: %w", err)
	}
	defer adminDB.Close()
	if err := grantExistingObjects(ctx, adminDB, cfg.RuntimeUser); err != nil {
		return Result{}, fmt.Errorf("grant existing objects: %w", err)
	}

	runtimeDB, err := open(ctx, dsn(cfg.RuntimeUser, cfg.RuntimePassword, cfg, cfg.DatabaseName))
	if err != nil {
		return Result{}, fmt.Errorf("validate runtime login: %w", err)
	}
	defer runtimeDB.Close()

	status, err := user.BootstrapFirstAdmin(user.NewStore(runtimeDB), cfg.FirstAdmin, user.BootstrapDeps{})
	if err != nil {
		return Result{}, fmt.Errorf("bootstrap first admin: %w", err)
	}
	return Result{FirstAdminStatus: string(status)}, nil
}

func ReadMigrationState(ctx context.Context, cfg Config) (MigrationState, error) {
	if err := cfg.Validate(); err != nil {
		return MigrationState{}, err
	}
	if _, err := migrationpolicy.LoadDirectory(cfg.MigrationsPath); err != nil {
		return MigrationState{}, fmt.Errorf("validate migration policy: %w", err)
	}

	database, err := open(ctx, dsn(cfg.MigrationUser, cfg.MigrationPassword, cfg, cfg.DatabaseName))
	if err != nil {
		return MigrationState{}, errors.New("connect to migration database for state inspection failed")
	}
	defer database.Close()

	var state MigrationState
	err = database.QueryRowContext(
		ctx,
		"SELECT version, dirty FROM public.schema_migrations LIMIT 1",
	).Scan(&state.Version, &state.Dirty)
	if errors.Is(err, sql.ErrNoRows) || sqlState(err) == "42P01" {
		return MigrationState{}, nil
	}
	if err != nil {
		return MigrationState{}, errors.New("read database migration state failed")
	}
	return state, nil
}

func open(ctx context.Context, connectionString string) (*sql.DB, error) {
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func dsn(username, password string, cfg Config, databaseName string) string {
	connectionURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(username, password),
		Host:   net.JoinHostPort(cfg.Host, cfg.Port),
		Path:   databaseName,
	}
	query := connectionURL.Query()
	query.Set("sslmode", cfg.SSLMode)
	connectionURL.RawQuery = query.Encode()
	return connectionURL.String()
}

func ensureDatabase(ctx context.Context, db *sql.DB, databaseName string) error {
	var exists bool
	if err := db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", databaseName).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err := db.ExecContext(ctx, "CREATE DATABASE "+quoteIdentifier(databaseName))
	return err
}

func reconcileRolesAndBaseGrants(ctx context.Context, db *sql.DB, cfg Config) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := reconcileRole(ctx, tx, cfg.MigrationUser, cfg.MigrationPassword); err != nil {
		return err
	}
	if err := reconcileRole(ctx, tx, cfg.RuntimeUser, cfg.RuntimePassword); err != nil {
		return err
	}

	statements := []string{
		"CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public",
		fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", quoteIdentifier(cfg.DatabaseName), quoteIdentifier(cfg.MigrationUser)),
		fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", quoteIdentifier(cfg.DatabaseName), quoteIdentifier(cfg.RuntimeUser)),
		fmt.Sprintf("GRANT USAGE, CREATE ON SCHEMA public TO %s", quoteIdentifier(cfg.MigrationUser)),
		fmt.Sprintf("ALTER SCHEMA public OWNER TO %s", quoteIdentifier(cfg.MigrationUser)),
		fmt.Sprintf("GRANT USAGE ON SCHEMA public TO %s", quoteIdentifier(cfg.RuntimeUser)),
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func reconcileRole(ctx context.Context, tx *sql.Tx, name, password string) error {
	var exists bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", name).Scan(&exists); err != nil {
		return err
	}
	verb := "CREATE ROLE"
	if exists {
		verb = "ALTER ROLE"
	}
	statement := fmt.Sprintf(
		"%s %s WITH LOGIN PASSWORD %s NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT",
		verb,
		quoteIdentifier(name),
		quoteLiteral(password),
	)
	_, err := tx.ExecContext(ctx, statement)
	return err
}

func applyDefaultPrivileges(ctx context.Context, db *sql.DB, cfg Config) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	statements := []string{
		fmt.Sprintf("ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %s", quoteIdentifier(cfg.MigrationUser), quoteIdentifier(cfg.RuntimeUser)),
		fmt.Sprintf("ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA public GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO %s", quoteIdentifier(cfg.MigrationUser), quoteIdentifier(cfg.RuntimeUser)),
		fmt.Sprintf("ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO %s", quoteIdentifier(cfg.MigrationUser), quoteIdentifier(cfg.RuntimeUser)),
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func grantExistingObjects(ctx context.Context, db *sql.DB, runtimeUser string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	statements := []string{
		fmt.Sprintf("GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %s", quoteIdentifier(runtimeUser)),
		fmt.Sprintf("GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO %s", quoteIdentifier(runtimeUser)),
		fmt.Sprintf("GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO %s", quoteIdentifier(runtimeUser)),
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func migrateUp(path, connectionString string, manifest migrationpolicy.Manifest) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	sourceURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(absPath)}).String()
	timeoutConnectionString, err := migrationConnectionString(connectionString)
	if err != nil {
		return err
	}
	migration, err := migrate.New(sourceURL, timeoutConnectionString)
	if err != nil {
		return errors.New("initialize migration driver failed")
	}
	defer migration.Close()
	current, dirty, err := migration.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		current = 0
		dirty = false
	} else if err != nil {
		return errors.New("read database migration state failed")
	}
	if err := manifest.ValidatePending(current, dirty); err != nil {
		return err
	}
	if err := migration.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return migrationExecutionError(err, migration)
	}
	return nil
}

type migrationStateReader interface {
	Version() (uint, bool, error)
}

func migrationExecutionError(cause error, reader migrationStateReader) error {
	reason := migrationFailureReason(cause)
	failedVersion, failedDirty, stateErr := reader.Version()
	if stateErr == nil {
		return fmt.Errorf(
			"migration execution failed: reason=%s version=%06d dirty=%t; inspect migration recovery guidance",
			reason,
			failedVersion,
			failedDirty,
		)
	}
	return fmt.Errorf(
		"migration execution failed: reason=%s; database migration state could not be read",
		reason,
	)
}

func migrationFailureReason(err error) string {
	if errors.Is(err, migrate.ErrLockTimeout) {
		return "migration_lock_timeout"
	}

	var databaseError migratedatabase.Error
	if errors.As(err, &databaseError) {
		err = databaseError.OrigErr
	} else {
		var databaseErrorPointer *migratedatabase.Error
		if errors.As(err, &databaseErrorPointer) {
			err = databaseErrorPointer.OrigErr
		}
	}

	switch sqlState(err) {
	case "55P03":
		return "lock_timeout"
	case "57014":
		return "statement_timeout"
	default:
		return "sql_error"
	}
}

func sqlState(err error) string {
	type sqlStateError interface {
		SQLState() string
	}
	var stateError sqlStateError
	if errors.As(err, &stateError) {
		return stateError.SQLState()
	}
	return ""
}

func migrationConnectionString(connectionString string) (string, error) {
	connectionURL, err := url.Parse(connectionString)
	if err != nil || connectionURL.Scheme == "" || connectionURL.Host == "" {
		return "", errors.New("migration database URL is invalid")
	}
	query := connectionURL.Query()
	options := strings.TrimSpace(query.Get("options"))
	options = strings.TrimSpace(options + " -c lock_timeout=5s -c statement_timeout=240s")
	query.Set("options", options)
	connectionURL.RawQuery = query.Encode()
	return connectionURL.String(), nil
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func quoteLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}

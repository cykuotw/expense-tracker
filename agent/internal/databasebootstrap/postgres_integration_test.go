package databasebootstrap

import (
	"database/sql"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

// Locate the command-owned assets independently of the test binary's cwd.
func migrationFiles(t *testing.T) fs.FS {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return os.DirFS(filepath.Join(filepath.Dir(file), "../../cmd/db-bootstrap"))
}

func TestPostgresBootstrapRoleAndReapplication(t *testing.T) {
	db := migrationDatabase(t)
	_, err := db.ExecContext(t.Context(), `CREATE TABLE public.backend_private(value TEXT);
		CREATE SCHEMA other_agent;
		CREATE TABLE other_agent.sessions(value TEXT);
		INSERT INTO other_agent.sessions VALUES('other agent state')`)
	require.NoError(t, err)
	role := "agent_state_test_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	require.NoError(t, Run(t.Context(), db, migrationFiles(t), role, "synthetic_test_password"))
	t.Cleanup(func() {
		_, err := db.Exec(`DROP OWNED BY "` + role + `"; DROP ROLE "` + role + `"`)
		require.NoError(t, err)
	})
	require.NoError(t, Run(t.Context(), db, migrationFiles(t), role, "synthetic_test_password"))
	var otherState string
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT value FROM other_agent.sessions`).Scan(&otherState))
	require.Equal(t, "other agent state", otherState)
	conn, err := db.Conn(t.Context())
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.ExecContext(t.Context(), `SET ROLE "`+role+`"`)
	require.NoError(t, err)
	defer conn.ExecContext(t.Context(), `RESET ROLE`)
	_, err = conn.ExecContext(t.Context(), `INSERT INTO expense_agent.sessions(app_name,owner_id,session_id) VALUES('bootstrap','owner','00000000-0000-4000-8000-000000000001')`)
	require.NoError(t, err)
	_, err = conn.ExecContext(t.Context(), `UPDATE expense_agent.sessions SET state='{"step":1}' WHERE app_name='bootstrap'`)
	require.NoError(t, err)
	var step int
	require.NoError(t, conn.QueryRowContext(t.Context(), `SELECT (state->>'step')::int FROM expense_agent.sessions WHERE app_name='bootstrap'`).Scan(&step))
	require.Equal(t, 1, step)
	for _, statement := range []string{
		`SELECT * FROM public.backend_private`,
		`SELECT * FROM other_agent.sessions`,
		`SELECT * FROM public.expense_agent_schema_migrations`,
		`CREATE TABLE expense_agent.forbidden(id INT)`,
		`DELETE FROM expense_agent.sessions`,
	} {
		_, err := conn.ExecContext(t.Context(), statement)
		require.ErrorContains(t, err, "permission denied")
	}
}

func migrationDatabase(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("AGENT_MIGRATION_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires test-only PostgreSQL operator URL with CREATEDB")
	}
	baseURL, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Contains(t, []string{"postgres", "postgresql"}, baseURL.Scheme)
	admin, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	name := "agent_migration_test_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	_, err = admin.ExecContext(t.Context(), `CREATE DATABASE "`+name+`"`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := admin.Exec(`DROP DATABASE "` + name + `" WITH (FORCE)`)
		require.NoError(t, err)
	})
	baseURL.Path = "/" + name
	baseURL.RawPath = ""
	db, err := sql.Open("postgres", baseURL.String())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func TestPostgresMigrationIsolationAndReapplication(t *testing.T) {
	db := migrationDatabase(t)
	_, err := db.ExecContext(t.Context(), `CREATE TABLE public.schema_migrations(version BIGINT PRIMARY KEY, dirty BOOLEAN NOT NULL);
		INSERT INTO public.schema_migrations VALUES(43, false)`)
	require.NoError(t, err)
	require.NoError(t, migrateUp(t.Context(), db, migrationFiles(t)))
	var tables int
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT count(*) FROM information_schema.tables WHERE table_schema='expense_agent' AND table_type='BASE TABLE'`).Scan(&tables))
	require.Equal(t, 3, tables)
	_, err = db.ExecContext(t.Context(), `INSERT INTO expense_agent.sessions(app_name,owner_id,session_id) VALUES('test','owner','00000000-0000-4000-8000-000000000002')`)
	require.NoError(t, err)
	require.NoError(t, migrateUp(t.Context(), db, migrationFiles(t)))
	var backendVersion, agentVersion, sessions int
	var backendDirty, agentDirty bool
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT version,dirty FROM public.schema_migrations`).Scan(&backendVersion, &backendDirty))
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT version,dirty FROM public.expense_agent_schema_migrations`).Scan(&agentVersion, &agentDirty))
	require.Equal(t, 43, backendVersion)
	require.False(t, backendDirty)
	require.Equal(t, 1, agentVersion)
	require.False(t, agentDirty)
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT count(*) FROM expense_agent.sessions`).Scan(&sessions))
	require.Equal(t, 1, sessions)
	_, err = db.ExecContext(t.Context(), `UPDATE public.expense_agent_schema_migrations SET dirty=true`)
	require.NoError(t, err)
	require.Error(t, migrateUp(t.Context(), db, migrationFiles(t)))
}

func TestPostgresMigrationRefusesExistingSchema(t *testing.T) {
	db := migrationDatabase(t)
	_, err := db.ExecContext(t.Context(), `CREATE SCHEMA expense_agent;
		CREATE TABLE expense_agent.retained(value TEXT);
		INSERT INTO expense_agent.retained VALUES('keep')`)
	require.NoError(t, err)
	require.ErrorContains(t, migrateUp(t.Context(), db, migrationFiles(t)), "untracked existing expense_agent schema")
	var retained string
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT value FROM expense_agent.retained`).Scan(&retained))
	require.Equal(t, "keep", retained)
}

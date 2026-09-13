package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type balanceLedgerErrorDB struct {
	err error
}

func (db balanceLedgerErrorDB) Exec(string, ...any) (sql.Result, error) {
	return nil, db.err
}

func (balanceLedgerErrorDB) Query(string, ...any) (*sql.Rows, error) {
	return nil, errors.New("unexpected query")
}

func TestCreateBalanceLedgerMapsNamedUniqueConstraint(t *testing.T) {
	store := &Store{db: balanceLedgerErrorDB{err: &pgconn.PgError{
		Code:           "23505",
		ConstraintName: balanceLedgerUniqueConstraint,
	}}}

	err := store.CreateBalanceLedger([]uuid.UUID{uuid.New()}, []uuid.UUID{uuid.New()})

	require.ErrorIs(t, err, types.ErrBalanceLedgerConflict)
}

func TestCreateBalanceLedgerPreservesOtherUniqueViolations(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505", ConstraintName: "another_constraint"}
	store := &Store{db: balanceLedgerErrorDB{err: pgErr}}

	err := store.CreateBalanceLedger([]uuid.UUID{uuid.New()}, []uuid.UUID{uuid.New()})

	require.ErrorIs(t, err, pgErr)
	require.NotErrorIs(t, err, types.ErrBalanceLedgerConflict)
}

func TestCreateBalanceLedgerMapsPostgresConstraint(t *testing.T) {
	database, err := dbstore.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		requireBalanceLedgerTestDB(t, "connect", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.Ping(); err != nil {
		requireBalanceLedgerTestDB(t, "ping", err)
	}

	tx, err := database.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	schema := strings.ReplaceAll("balance_ledger_store_"+uuid.NewString(), "-", "_")
	_, err = tx.Exec(fmt.Sprintf(`
		CREATE SCHEMA %s;
		SET LOCAL search_path TO %s;
		CREATE TABLE balance_ledger (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			balance_id UUID NOT NULL,
			ledger_id UUID NOT NULL
		);`, schema, schema))
	require.NoError(t, err)

	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migration, err := os.ReadFile(filepath.Join(
		filepath.Dir(sourceFile),
		"../../../cmd/migrate/migrations/000034_enforce_balance_ledger_uniqueness.up.sql",
	))
	require.NoError(t, err)
	_, err = tx.Exec(string(migration))
	require.NoError(t, err)

	store := &Store{db: tx, transactionBound: true}
	balanceID := uuid.New()
	ledgerID := uuid.New()
	require.NoError(t, store.CreateBalanceLedger([]uuid.UUID{balanceID}, []uuid.UUID{ledgerID}))
	require.ErrorIs(t, store.CreateBalanceLedger([]uuid.UUID{balanceID}, []uuid.UUID{ledgerID}), types.ErrBalanceLedgerConflict)
}

func requireBalanceLedgerTestDB(t *testing.T, operation string, err error) {
	t.Helper()
	if os.Getenv("EXPENSE_TRACKER_REQUIRE_TEST_DB") == "1" {
		t.Fatalf("%s required integration database: %v", operation, err)
	}
	t.Skipf("skipping: db %s error: %v", operation, err)
}

package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

var errLedgerRowsIteration = errors.New("ledger rows iteration failed")
var registerLedgerRowsErrorDriver sync.Once

type ledgerRowsErrorDriver struct{}

func (ledgerRowsErrorDriver) Open(string) (driver.Conn, error) {
	return ledgerRowsErrorConn{}, nil
}

type ledgerRowsErrorConn struct{}

func (ledgerRowsErrorConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (ledgerRowsErrorConn) Close() error {
	return nil
}

func (ledgerRowsErrorConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (ledgerRowsErrorConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return ledgerRowsErrorRows{}, nil
}

type ledgerRowsErrorRows struct{}

func (ledgerRowsErrorRows) Columns() []string {
	return []string{"id", "expense_id", "lender_user_id", "borrower_user_id", "share"}
}

func (ledgerRowsErrorRows) Close() error {
	return nil
}

func (ledgerRowsErrorRows) Next([]driver.Value) error {
	return errLedgerRowsIteration
}

func TestGetLedgerUnsettledFromGroupReturnsRowsError(t *testing.T) {
	const driverName = "expense-ledger-rows-error"
	registerLedgerRowsErrorDriver.Do(func() {
		sql.Register(driverName, ledgerRowsErrorDriver{})
	})

	db, err := sql.Open(driverName, "")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	store := NewStore(db)
	_, err = store.GetLedgerUnsettledFromGroup("group-id")

	require.ErrorIs(t, err, errLedgerRowsIteration)
}

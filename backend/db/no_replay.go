package db

import (
	"context"
	"database/sql/driver"
	"errors"

	"expense-tracker/backend/internal/dbavailability"
	"github.com/jackc/pgx/v5/stdlib"
)

// Once an operation has entered the driver, conservatively return failures to
// the caller. ErrBadConn otherwise makes database/sql transparently replay it,
// and the caller cannot establish whether a write reached PostgreSQL.
func noReplayError(err error) error {
	if errors.Is(err, driver.ErrBadConn) {
		return dbavailability.ErrUnavailable
	}
	return err
}

type noReplayConnector struct {
	driver.Connector
	monitor *dbavailability.Monitor
}

func (c noReplayConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &noReplayConn{Conn: conn.(*stdlib.Conn), monitor: c.monitor}, nil
}

type noReplayConn struct {
	*stdlib.Conn
	monitor *dbavailability.Monitor
}

func (c *noReplayConn) available(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.Conn.Conn().IsClosed() {
		c.monitor.Invalidate(true)
		return dbavailability.ErrUnavailable
	}
	return nil
}

func (c *noReplayConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if err := c.available(ctx); err != nil {
		return nil, err
	}
	result, err := c.Conn.ExecContext(ctx, query, args)
	return result, noReplayError(err)
}
func (c *noReplayConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if err := c.available(ctx); err != nil {
		return nil, err
	}
	rows, err := c.Conn.QueryContext(ctx, query, args)
	return rows, noReplayError(err)
}
func (c *noReplayConn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}
func (c *noReplayConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if err := c.available(ctx); err != nil {
		return nil, err
	}
	stmt, err := c.Conn.PrepareContext(ctx, query)
	if err != nil {
		return nil, noReplayError(err)
	}
	return &noReplayStmt{Stmt: stmt.(*stdlib.Stmt), conn: c}, nil
}

type noReplayStmt struct {
	*stdlib.Stmt
	conn *noReplayConn
}

func (s *noReplayStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if err := s.conn.available(ctx); err != nil {
		return nil, err
	}
	result, err := s.Stmt.ExecContext(ctx, args)
	return result, noReplayError(err)
}
func (s *noReplayStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if err := s.conn.available(ctx); err != nil {
		return nil, err
	}
	rows, err := s.Stmt.QueryContext(ctx, args)
	return rows, noReplayError(err)
}

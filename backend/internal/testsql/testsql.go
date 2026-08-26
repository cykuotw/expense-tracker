package testsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

const driverName = "expense-tracker-testsql"

var (
	registerDriver sync.Once
	scriptID       atomic.Uint64
	scripts        sync.Map
)

// Result describes one response from a database query.
type Result struct {
	Columns      []string
	Rows         [][]driver.Value
	QueryErr     error
	IterationErr error
	OnRowsClose  func()
}

type script struct {
	mu      sync.Mutex
	results []Result
	next    int
}

// Open returns a database that serves the supplied query results in order.
func Open(results ...Result) (*sql.DB, func()) {
	registerDriver.Do(func() {
		sql.Register(driverName, scriptedDriver{})
	})

	id := fmt.Sprintf("script-%d", scriptID.Add(1))
	scripts.Store(id, &script{results: results})
	db, err := sql.Open(driverName, id)
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)

	cleanup := func() {
		_ = db.Close()
		scripts.Delete(id)
	}
	return db, cleanup
}

type scriptedDriver struct{}

func (scriptedDriver) Open(name string) (driver.Conn, error) {
	value, ok := scripts.Load(name)
	if !ok {
		return nil, fmt.Errorf("testsql: script %q not found", name)
	}
	return &scriptedConn{script: value.(*script)}, nil
}

type scriptedConn struct {
	script *script
}

func (*scriptedConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("testsql: prepare is not supported")
}

func (*scriptedConn) Close() error {
	return nil
}

func (*scriptedConn) Begin() (driver.Tx, error) {
	return nil, errors.New("testsql: transactions are not supported")
}

func (c *scriptedConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	c.script.mu.Lock()
	defer c.script.mu.Unlock()

	if c.script.next >= len(c.script.results) {
		return nil, errors.New("testsql: unexpected query")
	}
	result := c.script.results[c.script.next]
	c.script.next++
	if result.QueryErr != nil {
		return nil, result.QueryErr
	}
	return &scriptedRows{result: result}, nil
}

type scriptedRows struct {
	result       Result
	next         int
	iterationErr bool
	closeOnce    sync.Once
}

func (r *scriptedRows) Columns() []string {
	return r.result.Columns
}

func (r *scriptedRows) Close() error {
	r.closeOnce.Do(func() {
		if r.result.OnRowsClose != nil {
			r.result.OnRowsClose()
		}
	})
	return nil
}

func (r *scriptedRows) Next(dest []driver.Value) error {
	if r.next < len(r.result.Rows) {
		row := r.result.Rows[r.next]
		r.next++
		if len(row) != len(dest) {
			return fmt.Errorf("testsql: row has %d values for %d columns", len(row), len(dest))
		}
		copy(dest, row)
		return nil
	}
	if r.result.IterationErr != nil && !r.iterationErr {
		r.iterationErr = true
		return r.result.IterationErr
	}
	return io.EOF
}

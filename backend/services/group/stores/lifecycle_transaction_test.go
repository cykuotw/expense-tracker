package group

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"expense-tracker/backend/types"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type lifecycleScript struct {
	active, settled, absent                 bool
	beginErr, queryErr, writeErr, commitErr error
	queries                                 []string
	writes, commits, rollbacks              int
}

type lifecycleConnector struct{ script *lifecycleScript }

func (c lifecycleConnector) Connect(context.Context) (driver.Conn, error) {
	return &lifecycleConnection{script: c.script}, nil
}
func (c lifecycleConnector) Driver() driver.Driver { return lifecycleDriver{} }

type lifecycleDriver struct{}

func (lifecycleDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }

type lifecycleConnection struct {
	script        *lifecycleScript
	inTransaction bool
}

func (*lifecycleConnection) Close() error { return nil }
func (*lifecycleConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unsupported prepare")
}
func (c *lifecycleConnection) Begin() (driver.Tx, error) {
	if c.script.beginErr != nil {
		return nil, c.script.beginErr
	}
	c.inTransaction = true
	return lifecycleTransaction{c.script}, nil
}
func (c *lifecycleConnection) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if !c.inTransaction {
		return nil, errors.New("query outside lifecycle transaction")
	}
	c.script.queries = append(c.script.queries, query)
	if c.script.queryErr != nil {
		return nil, c.script.queryErr
	}
	if len(c.script.queries) == 1 {
		if !strings.Contains(query, "FOR UPDATE") || !strings.Contains(query, "create_by_user_id = $2") || len(args) != 2 {
			return nil, errors.New("missing authorized group lock")
		}
		return &lifecycleRows{value: c.script.active, done: c.script.absent}, nil
	}
	return &lifecycleRows{value: c.script.settled}, nil
}
func (c *lifecycleConnection) ExecContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	if !c.inTransaction {
		return nil, errors.New("write outside lifecycle transaction")
	}
	c.script.writes++
	return driver.RowsAffected(1), c.script.writeErr
}

type lifecycleTransaction struct{ script *lifecycleScript }

func (tx lifecycleTransaction) Commit() error   { tx.script.commits++; return tx.script.commitErr }
func (tx lifecycleTransaction) Rollback() error { tx.script.rollbacks++; return nil }

type lifecycleRows struct {
	value bool
	done  bool
}

func (*lifecycleRows) Columns() []string { return []string{"value"} }
func (*lifecycleRows) Close() error      { return nil }
func (r *lifecycleRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	dest[0] = r.value
	r.done = true
	return nil
}

func TestLifecycleTransitionTransaction(t *testing.T) {
	failure := errors.New("database failure")
	for _, test := range []struct {
		name                     string
		script                   lifecycleScript
		desired                  bool
		want                     error
		writes, queries, commits int
	}{
		{name: "archive settled", script: lifecycleScript{active: true, settled: true}, writes: 1, queries: 2, commits: 1},
		{name: "archive conflict", script: lifecycleScript{active: true}, want: types.ErrGroupUnsettled, queries: 2},
		{name: "already archived", queries: 1, commits: 1},
		{name: "restore", desired: true, writes: 1, queries: 1, commits: 1},
		{name: "already active", script: lifecycleScript{active: true}, desired: true, queries: 1, commits: 1},
		{name: "unauthorized or absent", script: lifecycleScript{absent: true}, want: types.ErrGroupNotExist, queries: 1},
		{name: "begin failure", script: lifecycleScript{beginErr: failure}, want: failure},
		{name: "query failure", script: lifecycleScript{queryErr: failure}, want: failure, queries: 1},
		{name: "write failure", script: lifecycleScript{active: true, settled: true, writeErr: failure}, want: failure, writes: 1, queries: 2},
		{name: "commit failure", script: lifecycleScript{active: true, settled: true, commitErr: failure}, want: failure, writes: 1, queries: 2, commits: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			script := &test.script
			db := sql.OpenDB(lifecycleConnector{script})
			t.Cleanup(func() { _ = db.Close() })
			err := NewStore(db).UpdateGroupStatus("group", "creator", test.desired)
			if test.want == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, test.want)
			}
			require.Equal(t, test.writes, script.writes)
			require.Len(t, script.queries, test.queries)
			require.Equal(t, test.commits, script.commits)
			if test.want != nil && script.beginErr == nil && script.commits == 0 {
				require.Equal(t, 1, script.rollbacks)
			}
		})
	}
}

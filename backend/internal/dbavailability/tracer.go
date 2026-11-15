package dbavailability

import (
	"context"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ConnectionFailure distinguishes transport failures from statement errors.
// A deadline alone cannot tell us which operation timed out.
func ConnectionFailure(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	if pgerr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return strings.HasPrefix(pgerr.Code, "08") || pgerr.Code == "57P01" || pgerr.Code == "57P02" || pgerr.Code == "57P03"
	}
	_, ok := errors.AsType[*net.OpError](err)
	return ok
}

type operationKey struct{}
type probeKey struct{}
type operation struct {
	parent context.Context
	cancel context.CancelFunc
}

// ProbeContext suppresses operation-level observations while Check owns the
// overall probe result and cancellation. PingContext retains its total deadline.
func ProbeContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, probeKey{}, true)
}

type Tracer struct{ Monitor *Monitor }

func (t *Tracer) start(ctx context.Context, timeout time.Duration) context.Context {
	bounded, cancel := context.WithTimeout(ctx, timeout)
	return context.WithValue(bounded, operationKey{}, operation{ctx, cancel})
}

func (t *Tracer) end(ctx context.Context, err error, connecting bool) {
	op := ctx.Value(operationKey{}).(operation)
	defer op.cancel()
	if err == nil || errors.Is(op.parent.Err(), context.Canceled) || ctx.Value(probeKey{}) != nil {
		return
	}
	// Query deadlines include lock waits and slow SQL, even if cancellation also
	// closes the underlying connection. Do not turn those into a long outage.
	if !connecting && (errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded)) {
		t.Monitor.Invalidate(false)
		return
	}
	if connecting || ConnectionFailure(err) {
		t.Monitor.Invalidate(true)
		return
	}
	if pgerr, ok := errors.AsType[*pgconn.PgError](err); ok && pgerr.Code == "57014" {
		t.Monitor.Invalidate(false)
	}
}

func (t *Tracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	return t.start(ctx, OperationTimeout)
}
func (t *Tracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	if data.Err != nil && conn != nil && conn.IsClosed() {
		data.Err = errors.Join(data.Err, driver.ErrBadConn)
	}
	t.end(ctx, data.Err, false)
}
func (t *Tracer) TraceConnectStart(ctx context.Context, _ pgx.TraceConnectStartData) context.Context {
	return t.start(ctx, t.Monitor.ProbeTimeout())
}
func (t *Tracer) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	t.end(ctx, data.Err, true)
}
func (t *Tracer) TracePrepareStart(ctx context.Context, _ *pgx.Conn, _ pgx.TracePrepareStartData) context.Context {
	return t.start(ctx, OperationTimeout)
}
func (t *Tracer) TracePrepareEnd(ctx context.Context, conn *pgx.Conn, data pgx.TracePrepareEndData) {
	if data.Err != nil && conn != nil && conn.IsClosed() {
		data.Err = errors.Join(data.Err, driver.ErrBadConn)
	}
	t.end(ctx, data.Err, false)
}

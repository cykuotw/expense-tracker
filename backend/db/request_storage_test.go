package db

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"expense-tracker/backend/config"
	"expense-tracker/backend/internal/dbavailability"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgproto3"
)

// Exercise the actual pgx/database/sql path without a database or a listening
// socket. The peer speaks only the startup and simple-query protocol we need.
func TestRequestPoolObservesShutdownAndDoesNotReplayWrite(t *testing.T) {
	for _, query := range []bool{false, true} {
		t.Run(fmt.Sprintf("query=%v", query), func(t *testing.T) { testRequestPoolShutdown(t, query) })
	}
}

func testRequestPoolShutdown(t *testing.T, query bool) {
	t.Helper()
	now := time.Now().UTC()
	policy, err := dbavailability.ParsePolicy(fmt.Sprintf(`{"enabled":true,"start":%q,"stop":%q,"timezone":"UTC"}`, now.Add(time.Hour).Format("15:04"), now.Add(2*time.Hour).Format("15:04")))
	if err != nil {
		t.Fatal(err)
	}
	monitor := dbavailability.New(policy)
	pgcfg, err := pgx.ParseConfig("postgres://fixture:fixture@127.0.0.1/fixture?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	pgcfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	var writes atomic.Int32
	pgcfg.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		client, server := net.Pipe()
		t.Cleanup(func() { _ = client.Close(); _ = server.Close() })
		go func() {
			defer server.Close()
			peer := pgproto3.NewBackend(server, server)
			if _, err := peer.ReceiveStartupMessage(); err != nil {
				return
			}
			peer.Send(&pgproto3.AuthenticationOk{})
			peer.Send(&pgproto3.ParameterStatus{Name: "server_version", Value: "16.0"})
			peer.Send(&pgproto3.ParameterStatus{Name: "client_encoding", Value: "UTF8"})
			peer.Send(&pgproto3.ParameterStatus{Name: "standard_conforming_strings", Value: "on"})
			peer.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
			if err := peer.Flush(); err != nil {
				return
			}
			for {
				message, err := peer.Receive()
				if err != nil {
					return
				}
				query, ok := message.(*pgproto3.Query)
				if !ok {
					return
				}
				if query.String == "UPDATE fixture SET value = 1" {
					// Connection is lost after accepting a write. Its outcome is
					// ambiguous, so neither pgx nor availability may replay it.
					writes.Add(1)
					// Reading a response byte confirms the client's write has
					// completed before simulating the transport loss.
					peer.Send(&pgproto3.NoticeResponse{Severity: "NOTICE", Code: "00000", Message: "accepted"})
					_ = peer.Flush()
					return
				}
				peer.Send(&pgproto3.EmptyQueryResponse{})
				peer.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
				if err := peer.Flush(); err != nil {
					return
				}
			}
		}()
		return client, nil
	}
	storage := newRequestStorage(config.Config{}, pgcfg, monitor)
	t.Cleanup(func() { _ = storage.Close() })
	if storage.Stats().OpenConnections != 0 {
		t.Fatal("pool connected during construction")
	}
	if err := monitor.Check(t.Context(), false); err != nil {
		t.Fatal(err)
	}
	if query {
		rows, err := storage.QueryContext(t.Context(), "UPDATE fixture SET value = 1")
		if rows != nil {
			_ = rows.Close()
		}
		if err == nil {
			t.Fatal("expected connection failure")
		}
	} else if _, err := storage.ExecContext(t.Context(), "UPDATE fixture SET value = 1"); err == nil {
		t.Fatal("expected connection failure")
	}
	if writes.Load() != 1 {
		t.Fatalf("ambiguous write replayed %d times", writes.Load())
	}
	if err := monitor.Check(t.Context(), false); !errors.Is(err, dbavailability.ErrUnavailable) {
		t.Fatalf("shutdown not cached: %v", err)
	}
}

func TestRequestPoolConnectHonorsProbeCancellation(t *testing.T) {
	monitor := dbavailability.New(dbavailability.Policy{})
	pgcfg, err := pgx.ParseConfig("postgres://fixture:fixture@127.0.0.1/fixture?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	pgcfg.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) { <-ctx.Done(); return nil, ctx.Err() }
	storage := newRequestStorage(config.Config{}, pgcfg, monitor)
	t.Cleanup(func() { _ = storage.Close() })
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if err := monitor.Check(ctx, false); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("dial did not honor cancellation: %v", err)
	}
}

func TestRequestPoolUsesConfiguredConnectionDeadline(t *testing.T) {
	monitor, err := dbavailability.NewWithDurations(dbavailability.Policy{}, 8*time.Second, 90*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	pgcfg, err := pgx.ParseConfig("postgres://fixture:fixture@127.0.0.1/fixture?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	dialed := false
	pgcfg.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
		dialed = true
		deadline, ok := ctx.Deadline()
		if remaining := time.Until(deadline); !ok || remaining > 8*time.Second || remaining < 7*time.Second {
			t.Fatalf("configured connect deadline not used: %v", remaining)
		}
		return nil, errors.New("fixture connection failure")
	}
	storage := newRequestStorage(config.Config{}, pgcfg, monitor)
	t.Cleanup(func() { _ = storage.Close() })
	if err := monitor.Check(t.Context(), false); !errors.Is(err, dbavailability.ErrUnavailable) || !dialed {
		t.Fatalf("expected a bounded dial attempt: %v", err)
	}
}

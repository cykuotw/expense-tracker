package dbavailability

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func fixture(t *testing.T, instant string) (*Monitor, *time.Time) {
	t.Helper()
	p, err := ParsePolicy(`{"enabled":true,"start":"09:00","stop":"22:00","timezone":"America/Toronto"}`)
	if err != nil {
		t.Fatal(err)
	}
	now, err := time.Parse(time.RFC3339, instant)
	if err != nil {
		t.Fatal(err)
	}
	m := New(p)
	m.now = func() time.Time { return now }
	return m, &now
}

func TestFailureCacheAndManualRecovery(t *testing.T) {
	m, now := fixture(t, "2026-11-15T04:00:00Z")
	calls := 0
	m.SetProbe(func(ctx context.Context) error {
		calls++
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > DefaultProbeTimeout {
			t.Error("missing total probe deadline")
		}
		return io.EOF
	})
	if !errors.Is(m.Check(t.Context(), false), ErrUnavailable) {
		t.Fatal("expected failure")
	}
	for range 3 {
		_ = m.Check(t.Context(), false)
	}
	if calls != 1 {
		t.Fatalf("cache did not suppress probes: %d", calls)
	}
	*now = now.Add(DefaultFailureTTL)
	if calls != 1 {
		t.Fatal("expiry triggered work")
	}
	m.SetProbe(func(context.Context) error { calls++; return nil })
	if err := m.Check(t.Context(), false); err != nil {
		t.Fatal(err)
	}
	if err := m.Check(t.Context(), false); err != nil || calls != 2 {
		t.Fatal("healthy requests should reuse availability")
	}
	// EC2 goes down after successful recovery; the next observed failure wins.
	m.Invalidate(true)
	if !errors.Is(m.Check(t.Context(), false), ErrUnavailable) || calls != 2 {
		t.Fatal("connection failure did not replace healthy state")
	}
}

func TestRetryCooldown(t *testing.T) {
	m, now := fixture(t, "2026-11-15T04:00:00Z")
	calls := 0
	m.SetProbe(func(context.Context) error { calls++; return io.EOF })
	_ = m.Check(t.Context(), false)
	_ = m.Check(t.Context(), true)
	_ = m.Check(t.Context(), true)
	if calls != 2 {
		t.Fatal("explicit retry was not throttled")
	}
	*now = now.Add(RetryCooldown)
	m.SetProbe(func(context.Context) error { calls++; return nil })
	if err := m.Check(t.Context(), true); err != nil || calls != 3 {
		t.Fatal("manual startup did not recover through retry")
	}
}

func TestEntryIntoOperatingHoursAndInHoursRecovery(t *testing.T) {
	m, now := fixture(t, "2026-11-15T13:58:37Z")
	calls := 0
	m.SetProbe(func(context.Context) error { calls++; return io.EOF })
	_ = m.Check(t.Context(), false)
	if !m.until.Equal(time.Date(2026, 11, 15, 14, 0, 0, 0, time.UTC)) {
		t.Fatalf("failure crosses opening: %v", m.until)
	}
	*now = time.Date(2026, 11, 15, 14, 0, 0, 0, time.UTC)
	_ = m.Check(t.Context(), false)
	_ = m.Check(t.Context(), false)
	if calls != 3 {
		t.Fatal("in-hours requests should reattempt without five-minute cache")
	}
}

func TestProbeCancellationDoesNotPoisonCache(t *testing.T) {
	m, _ := fixture(t, "2026-11-15T04:00:00Z")
	ctx, cancel := context.WithCancel(t.Context())
	m.SetProbe(func(ctx context.Context) error { cancel(); <-ctx.Done(); return ctx.Err() })
	if !errors.Is(m.Check(ctx, false), context.Canceled) {
		t.Fatal("cancellation not returned")
	}
	if !m.until.IsZero() {
		t.Fatal("caller cancellation cached as outage")
	}
	m.SetProbe(func(context.Context) error { return nil })
	if err := m.Check(t.Context(), false); err != nil {
		t.Fatal(err)
	}
}

func TestProbeHonorsEarlierDeadline(t *testing.T) {
	m, _ := fixture(t, "2026-11-15T04:00:00Z")
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	m.SetProbe(func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() })
	if !errors.Is(m.Check(ctx, false), context.DeadlineExceeded) {
		t.Fatal("earlier request deadline ignored")
	}
	if !m.until.IsZero() {
		t.Fatal("request cancellation cached as outage")
	}
}

func TestConcurrentRequestsAndStaleSuccess(t *testing.T) {
	m, _ := fixture(t, "2026-11-15T04:00:00Z")
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	m.SetProbe(func(context.Context) error { calls.Add(1); close(started); <-release; return nil })
	var wg sync.WaitGroup
	wg.Go(func() { _ = m.Check(t.Context(), false) })
	<-started
	// An independent operation detects shutdown while the probe is in flight.
	m.Invalidate(true)
	for range 10 {
		wg.Go(func() {
			if !errors.Is(m.Check(t.Context(), false), ErrUnavailable) {
				t.Error("stale probe success won")
			}
		})
	}
	close(release)
	wg.Wait()
	if calls.Load() != 1 || m.ready {
		t.Fatal("concurrent probe or stale success")
	}
}

func TestWaiterCancellationDoesNotCancelSharedProbe(t *testing.T) {
	m, _ := fixture(t, "2026-11-15T04:00:00Z")
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	m.SetProbe(func(context.Context) error { close(started); <-release; return nil })
	go func() { done <- m.Check(t.Context(), false) }()
	<-started
	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()
	if !errors.Is(m.Check(ctx, false), context.DeadlineExceeded) {
		t.Fatal("waiter did not cancel")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestFreshEnvironmentAndHealthyOffHours(t *testing.T) {
	for range 2 {
		m, _ := fixture(t, "2026-11-15T04:00:00Z")
		calls := 0
		m.SetProbe(func(context.Context) error { calls++; return nil })
		if err := m.Check(t.Context(), false); err != nil || calls != 1 {
			t.Fatal("new environment must probe and allow manual startup")
		}
	}
}

func TestPolicyOvernightAndDST(t *testing.T) {
	for _, tc := range []struct {
		raw, instant string
		off          bool
	}{
		{`{"enabled":true,"start":"22:00","stop":"05:00","timezone":"UTC"}`, "2026-11-15T01:00:00Z", false},
		{`{"enabled":true,"start":"22:00","stop":"05:00","timezone":"UTC"}`, "2026-11-15T05:00:00Z", true},
		{`{"enabled":true,"start":"09:00","stop":"22:00","timezone":"America/Toronto"}`, "2026-03-08T13:00:00Z", false},
		{`{"enabled":true,"start":"09:00","stop":"22:00","timezone":"America/Toronto"}`, "2026-11-01T14:00:00Z", false},
	} {
		p, err := ParsePolicy(tc.raw)
		if err != nil {
			t.Fatal(err)
		}
		now, _ := time.Parse(time.RFC3339, tc.instant)
		if p.offHours(now) != tc.off {
			t.Fatalf("incorrect window: %s", tc.instant)
		}
	}
	for _, raw := range []string{`{`, `{"enabled":true,"timezone":"Unknown"}`, `{"enabled":true,"timezone":"UTC","start":"09:00","stop":"09:00"}`} {
		if _, err := ParsePolicy(raw); err == nil {
			t.Fatal("invalid policy accepted")
		}
	}
}

func TestFailureExpiryAtDSTOpening(t *testing.T) {
	for _, tc := range []struct{ start, stop, instant, expiry string }{
		{"03:00", "05:00", "2026-03-08T06:58:37Z", "2026-03-08T07:00:00Z"},
		{"01:00", "03:00", "2026-11-01T04:58:37Z", "2026-11-01T05:00:00Z"},
	} {
		p, err := ParsePolicy(`{"enabled":true,"start":"` + tc.start + `","stop":"` + tc.stop + `","timezone":"America/Toronto"}`)
		if err != nil {
			t.Fatal(err)
		}
		now, _ := time.Parse(time.RFC3339, tc.instant)
		expiry, _ := time.Parse(time.RFC3339, tc.expiry)
		if !p.failureExpiry(now, DefaultFailureTTL).Equal(expiry) {
			t.Fatalf("incorrect DST expiry: %v", p.failureExpiry(now, DefaultFailureTTL))
		}
	}
}

func TestConfiguredDurationsControlProbeCacheAndOpening(t *testing.T) {
	base, now := fixture(t, "2026-11-15T04:00:00Z")
	m, err := NewWithDurations(base.policy, 8*time.Second, 90*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	m.now = base.now
	calls := 0
	m.SetProbe(func(ctx context.Context) error {
		calls++
		deadline, ok := ctx.Deadline()
		remaining := time.Until(deadline)
		if !ok || remaining > 8*time.Second || remaining < 7*time.Second {
			t.Fatalf("configured deadline not used: %v", remaining)
		}
		return io.EOF
	})
	_ = m.Check(t.Context(), false)
	if !m.until.Equal(now.Add(90 * time.Second)) {
		t.Fatalf("configured cache not used: %v", m.until)
	}
	*now = now.Add(89 * time.Second)
	_ = m.Check(t.Context(), false)
	if calls != 1 {
		t.Fatal("cache expired early")
	}
	*now = now.Add(time.Second)
	_ = m.Check(t.Context(), false)
	if calls != 2 {
		t.Fatal("configured expiry did not permit next probe")
	}
	// Observed connection failures must use the same configured cache duration.
	m.Invalidate(true)
	if !m.until.Equal(now.Add(90 * time.Second)) {
		t.Fatal("connection failure used a different cache duration")
	}
	*now = time.Date(2026, 11, 15, 13, 58, 30, 0, time.UTC)
	longCache, err := NewWithDurations(base.policy, 8*time.Second, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	longCache.now = base.now
	longCache.Invalidate(true)
	if !longCache.until.Equal(time.Date(2026, 11, 15, 14, 0, 0, 0, time.UTC)) {
		t.Fatal("long cache crossed opening time")
	}
}

func TestConfiguredProbeDoesNotChangeSQLDeadline(t *testing.T) {
	m, err := NewWithDurations(Policy{}, 9*time.Second, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	tracer := &Tracer{Monitor: m}
	for _, tc := range []struct {
		name     string
		ctx      context.Context
		expected time.Duration
	}{
		{"connect", tracer.TraceConnectStart(t.Context(), pgx.TraceConnectStartData{}), 9 * time.Second},
		{"query", tracer.TraceQueryStart(t.Context(), nil, pgx.TraceQueryStartData{}), OperationTimeout},
		{"prepare", tracer.TracePrepareStart(t.Context(), nil, pgx.TracePrepareStartData{}), OperationTimeout},
	} {
		deadline, ok := tc.ctx.Deadline()
		if remaining := time.Until(deadline); !ok || remaining > tc.expected || remaining < tc.expected-time.Second {
			t.Fatalf("%s deadline incorrect: %v", tc.name, remaining)
		}
		tracer.end(tc.ctx, nil, false)
	}
}

func TestConfiguredDurationBounds(t *testing.T) {
	for _, tc := range []struct{ probe, cache time.Duration }{
		{0, DefaultFailureTTL}, {11 * time.Second, DefaultFailureTTL},
		{DefaultProbeTimeout, 0}, {DefaultProbeTimeout, time.Hour + time.Second},
	} {
		if _, err := NewWithDurations(Policy{}, tc.probe, tc.cache); err == nil {
			t.Fatal("invalid durations accepted")
		}
	}
	for _, tc := range []struct{ probe, cache time.Duration }{{time.Second, time.Second}, {10 * time.Second, time.Hour}} {
		if _, err := NewWithDurations(Policy{}, tc.probe, tc.cache); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTracerClassifiesFailures(t *testing.T) {
	for _, tc := range []struct {
		name               string
		connecting, cancel bool
		err                error
		outage, invalidate bool
	}{
		{"connection refused", true, false, io.EOF, true, true},
		{"connection deadline", true, false, context.DeadlineExceeded, true, true},
		{"lost connection", false, false, io.EOF, true, true},
		{"SQL deadline", false, false, context.DeadlineExceeded, false, true},
		{"SQL statement timeout", false, false, &pgconn.PgError{Code: "57014"}, false, true},
		{"constraint violation", false, false, &pgconn.PgError{Code: "23505"}, false, false},
		{"caller cancel", false, true, context.Canceled, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := fixture(t, "2026-11-15T04:00:00Z")
			m.ready = true
			tracer := &Tracer{Monitor: m}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			bounded := tracer.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{})
			if tc.cancel {
				cancel()
			}
			tracer.end(bounded, tc.err, tc.connecting)
			if m.ready == tc.invalidate || (!m.until.IsZero()) != tc.outage {
				t.Fatalf("ready=%v until=%v", m.ready, m.until)
			}
		})
	}
}

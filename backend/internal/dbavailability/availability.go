// Package dbavailability coordinates request-driven database recovery. State is
// local to one process; it is not proof of the EC2 instance's power state.
package dbavailability

import (
	"context"
	"encoding/json/v2"
	"errors"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"
)

const (
	DefaultProbeTimeout = 5 * time.Second
	DefaultFailureTTL   = 5 * time.Minute
	OperationTimeout    = 5 * time.Second
	RetryCooldown       = 30 * time.Second
)

var ErrUnavailable = errors.New("database temporarily unavailable")

type Policy struct {
	Enabled         bool   `json:"enabled"`
	Start           string `json:"start"`
	Stop            string `json:"stop"`
	Timezone        string `json:"timezone"`
	zone            *time.Location
	start, duration int
}

func ParsePolicy(raw string) (Policy, error) {
	var p Policy
	if raw == "" {
		return p, nil
	}
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return p, errors.New("invalid database operating-hours policy")
	}
	if !p.Enabled {
		return p, nil
	}
	var err error
	if p.Timezone != "UTC" && !strings.Contains(p.Timezone, "/") {
		return p, errors.New("database operating-hours timezone must be an IANA zone")
	}
	p.zone, err = time.LoadLocation(p.Timezone)
	if err != nil {
		return p, errors.New("invalid database operating-hours timezone")
	}
	start, err := time.Parse("15:04", p.Start)
	if err != nil || start.Format("15:04") != p.Start {
		return p, errors.New("invalid database operating-hours start")
	}
	stop, err := time.Parse("15:04", p.Stop)
	if err != nil || stop.Format("15:04") != p.Stop {
		return p, errors.New("invalid database operating-hours stop")
	}
	p.start = start.Hour()*60 + start.Minute()
	p.duration = (stop.Hour()*60 + stop.Minute() - p.start + 1440) % 1440
	if p.duration == 0 {
		return p, errors.New("empty database operating-hours window")
	}
	return p, nil
}

func (p Policy) offHours(now time.Time) bool {
	if !p.Enabled {
		return false
	}
	local := now.In(p.zone)
	return (local.Hour()*60+local.Minute()-p.start+1440)%1440 >= p.duration
}

// Check only the cache's elapsed duration, so repeated/skipped wall times use
// the actual timezone rules rather than constructing an ambiguous local date.
func (p Policy) failureExpiry(now time.Time, ttl time.Duration) time.Time {
	if !p.offHours(now) {
		return time.Time{}
	}
	expiry := now.Add(ttl)
	for next := now.Truncate(time.Minute).Add(time.Minute); next.Before(expiry); next = next.Add(time.Minute) {
		if !p.offHours(next) {
			return next
		}
	}
	return expiry
}

type flight struct {
	done chan struct{}
	err  error
}

type Monitor struct {
	mu           sync.Mutex
	policy       Policy
	probeTimeout time.Duration
	failureTTL   time.Duration
	now          func() time.Time
	probe        func(context.Context) error
	ready        bool
	until        time.Time
	lastRetry    time.Time
	version      uint64
	flight       *flight
}

func New(policy Policy) *Monitor {
	return &Monitor{policy: policy, now: time.Now, probeTimeout: DefaultProbeTimeout, failureTTL: DefaultFailureTTL}
}

func NewWithDurations(policy Policy, probeTimeout, failureTTL time.Duration) (*Monitor, error) {
	if probeTimeout < time.Second || probeTimeout > 10*time.Second {
		return nil, errors.New("database probe timeout must be between 1 and 10 seconds")
	}
	if failureTTL < time.Second || failureTTL > time.Hour {
		return nil, errors.New("database off-hours failure cache must be between 1 and 3600 seconds")
	}
	m := New(policy)
	m.probeTimeout, m.failureTTL = probeTimeout, failureTTL
	return m, nil
}

// ProbeTimeout is immutable after assembly and also bounds connection setup.
func (m *Monitor) ProbeTimeout() time.Duration { return m.probeTimeout }

// SetProbe is called once during assembly, before serving requests.
func (m *Monitor) SetProbe(probe func(context.Context) error) { m.probe = probe }

// Invalidate never promotes availability. Only a successful explicit probe may
// do that, and only if no newer failure was observed while it was running.
func (m *Monitor) Invalidate(connectionFailure bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.version++
	m.ready = false
	if connectionFailure {
		m.until = m.policy.failureExpiry(m.now(), m.failureTTL)
	}
}

func (m *Monitor) Check(ctx context.Context, retry bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	if pending := m.flight; pending != nil {
		m.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pending.done:
			return pending.err
		}
	}
	now := m.now()
	if m.ready {
		m.mu.Unlock()
		return nil
	}
	if now.Before(m.until) {
		if !retry || (!m.lastRetry.IsZero() && now.Sub(m.lastRetry) < RetryCooldown) {
			m.mu.Unlock()
			return ErrUnavailable
		}
	}
	if retry {
		m.lastRetry = now
	}
	pending := &flight{done: make(chan struct{})}
	m.flight = pending
	version := m.version
	m.mu.Unlock()
	probeCtx, cancel := context.WithTimeout(ctx, m.probeTimeout)
	err := m.probe(probeCtx)
	if err == nil {
		err = probeCtx.Err()
	}
	cancel()
	m.mu.Lock()
	defer m.mu.Unlock()
	if ctx.Err() != nil {
		pending.err = ctx.Err()
	} else if err != nil {
		m.version++
		m.ready = false
		m.until = m.policy.failureExpiry(m.now(), m.failureTTL)
		pending.err = ErrUnavailable
	} else if version != m.version {
		pending.err = ErrUnavailable
	} else {
		m.ready = true
		m.until = time.Time{}
	}
	m.flight = nil
	close(pending.done)
	return pending.err
}

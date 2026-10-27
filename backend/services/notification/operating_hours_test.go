package notification

import (
	"strings"
	"testing"
	"time"
)

func TestSenderOperatingWindow(t *testing.T) {
	policy := `{"enabled":true,"start":"09:00","stop":"22:00","safety_stop":"01:00","timezone":"America/Toronto"}`
	for _, test := range []struct {
		instant string
		allowed bool
	}{
		{"2026-07-01T12:59:00Z", false},
		{"2026-07-01T13:00:00Z", true},
		{"2026-07-02T02:00:00Z", false},
		{"2026-01-01T14:00:00Z", true},
		{"2026-11-01T05:00:00Z", false},
		{"2026-11-01T06:00:00Z", false},
		{"2026-11-01T14:00:00Z", true},
	} {
		t.Run(test.instant, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, test.instant)
			if err != nil {
				t.Fatal(err)
			}
			allowed, err := SenderAllowed(strings.NewReader(policy), now)
			if err != nil || allowed != test.allowed {
				t.Fatalf("allowed=%v err=%v", allowed, err)
			}
		})
	}
	for _, policy := range []string{`{`, `{"enabled":true,"timezone":"unknown"}`, `{"enabled":true,"timezone":"UTC","start":"09:00","stop":"09:00"}`} {
		if _, err := SenderAllowed(strings.NewReader(policy), time.Now()); err == nil {
			t.Fatalf("accepted invalid policy %s", policy)
		}
	}
	allowed, err := SenderAllowed(strings.NewReader(`{"enabled":false}`), time.Now())
	if err != nil || !allowed {
		t.Fatalf("disabled policy changed existing behavior: %v", err)
	}
}

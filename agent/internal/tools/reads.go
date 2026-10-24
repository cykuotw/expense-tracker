package tools

import (
	"context"
	"sync"

	"google.golang.org/adk/v2/agent"
)

type readsKey struct{}
type reads struct {
	mu     sync.Mutex
	agents map[string]bool
}

// WithReadTracking tracks successful business reads within one invocation.
// It grants no actor identity or resource authorization.
func WithReadTracking(ctx context.Context) context.Context {
	return context.WithValue(ctx, readsKey{}, &reads{agents: map[string]bool{}})
}

func recordRead(ctx agent.Context) {
	if tracking, ok := ctx.Value(readsKey{}).(*reads); ok {
		tracking.mu.Lock()
		defer tracking.mu.Unlock()
		tracking.agents[ctx.AgentName()+"|"+ctx.Path()] = true
	}
}

func ReadSucceeded(ctx agent.Context) bool {
	tracking, ok := ctx.Value(readsKey{}).(*reads)
	if !ok {
		return false
	}
	tracking.mu.Lock()
	defer tracking.mu.Unlock()
	return tracking.agents[ctx.AgentName()+"|"+ctx.Path()]
}

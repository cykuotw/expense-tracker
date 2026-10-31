// Package chat manages authenticated chat turns and their application lifecycle.
package chat

import (
	"context"
	"fmt"
	"sync"

	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/runtime/engine"
	sessionstore "expense-tracker/agent/internal/store/session"
)

// Dependencies supplies an ADK engine and chat lifecycle services.
type Dependencies struct {
	Engine       *engine.Engine
	Records      sessionstore.Repository
	Authenticate func(context.Context) (models.Identity, error)
	Limits       models.LLMLimits
}

// Service owns authenticated chat turns, admission and optional recording.
type Service struct {
	engine         *engine.Engine
	records        sessionstore.Repository
	authenticate   func(context.Context) (models.Identity, error)
	limits         models.LLMLimits
	mu             sync.Mutex
	conversations  map[conversationKey]*conversationTracker
	activeRunCount int
}

// New assembles a chat service around an already-built ADK engine.
func New(deps Dependencies) (*Service, error) {
	if deps.Engine == nil {
		return nil, fmt.Errorf("explicit ADK engine required")
	}
	if deps.Records != nil && deps.Authenticate == nil {
		return nil, fmt.Errorf("durable execution requires explicit authentication")
	}
	if err := deps.Limits.Validate(); err != nil {
		return nil, err
	}
	return &Service{engine: deps.Engine, records: deps.Records, authenticate: deps.Authenticate,
		limits: deps.Limits, conversations: map[conversationKey]*conversationTracker{}}, nil
}

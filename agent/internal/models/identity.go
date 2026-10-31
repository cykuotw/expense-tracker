package models

import "time"

// Identity is a trusted authentication result, never a model-supplied argument.
type Identity struct {
	ID        string
	ExpiresAt time.Time
}

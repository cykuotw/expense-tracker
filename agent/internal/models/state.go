package models

// StateUpdate edits arbitrary top-level keys in the session's agent_context.
// Set replaces a whole value (including objects/arrays/null); Unset removes it.
type StateUpdate struct {
	Set   map[string]any `json:"set,omitempty"`
	Unset []string       `json:"unset,omitempty"`
}

type StateUpdateResult struct {
	Status string         `json:"status"`
	State  map[string]any `json:"state"`
}

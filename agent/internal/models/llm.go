package models

import (
	"fmt"
	"strings"
)

// ModelProfile binds a configured model to its output-token ceiling.
type ModelProfile struct {
	Name            string
	MaxOutputTokens int32
}

// ModelPolicy maps trusted skill roles to operator-configured model profiles.
type ModelPolicy struct {
	Default ModelProfile
	Router  ModelProfile
}

func (policy ModelPolicy) Validate() error {
	for _, profile := range []struct {
		name  string
		value ModelProfile
	}{
		{"default", policy.Default}, {"router", policy.Router},
	} {
		if strings.TrimSpace(profile.value.Name) == "" || profile.value.MaxOutputTokens <= 0 {
			return fmt.Errorf("%s model requires a name and positive output-token limit", profile.name)
		}
	}
	return nil
}

// LLMLimits contains immutable per-turn and per-model-call ceilings.
type LLMLimits struct {
	MaxModelCalls         int32
	MaxToolCalls          int32
	MaxStateUpdates       int32
	MaxContextBytes       int32
	MaxResponseBytes      int32
	MaxTotalResponseBytes int32
	MaxResponseEvents     int32
}

func (limits LLMLimits) Validate() error {
	for _, setting := range []struct {
		name  string
		value int32
	}{
		{"model calls", limits.MaxModelCalls}, {"tool calls", limits.MaxToolCalls},
		{"state updates", limits.MaxStateUpdates}, {"context bytes", limits.MaxContextBytes},
		{"response bytes", limits.MaxResponseBytes}, {"total response bytes", limits.MaxTotalResponseBytes},
		{"response events", limits.MaxResponseEvents},
	} {
		if setting.value <= 0 {
			return fmt.Errorf("%s limit must be positive", setting.name)
		}
	}
	if limits.MaxTotalResponseBytes < limits.MaxResponseBytes {
		return fmt.Errorf("total response bytes must be at least the per-response limit")
	}
	return nil
}

// ModelSelection describes the model and output ceiling chosen for a skill role.
type ModelSelection struct {
	Requested       string
	Selected        string
	MaxOutputTokens int32
	Fallback        bool
	Reason          string
}

// SkillMetadata identifies a bundled skill and its configured model.
type SkillMetadata struct {
	Name           string
	Version        string
	ModelSelection ModelSelection
}

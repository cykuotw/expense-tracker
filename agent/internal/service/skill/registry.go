package skill

import (
	"fmt"
	"slices"
	"strings"
)

type Registry struct {
	Router Skill
	Leaves []Skill
}

// NewRegistry validates bundled skills against tool names approved by the runtime.
// It does not construct agents or executable tools.
func NewRegistry(skills []Skill, allowedTools []string) (Registry, error) {
	var registry Registry
	names := map[string]bool{}
	for _, skill := range skills {
		name := AgentName(skill.Name)
		if names[name] {
			return registry, fmt.Errorf("reserved or duplicate agent name %q", skill.Name)
		}
		names[name] = true

		seenTools := map[string]bool{}
		for _, name := range skill.Tools {
			if !slices.Contains(allowedTools, name) || seenTools[name] {
				return registry, fmt.Errorf("invalid tool grant %q in %s", name, skill.Name)
			}
			seenTools[name] = true
		}

		if skill.Name == "orchestrator" {
			if len(skill.Tools) != 0 {
				return registry, fmt.Errorf("orchestrator cannot receive business tools")
			}
			registry.Router = skill
		} else {
			registry.Leaves = append(registry.Leaves, skill)
		}
	}
	if registry.Router.Name == "" || len(registry.Leaves) == 0 {
		return registry, fmt.Errorf("one orchestrator and at least one leaf skill required")
	}
	return registry, nil
}

// AgentName prefixes leaves so delegation names cannot collide with business tools.
func AgentName(skill string) string {
	if skill == "orchestrator" {
		return skill
	}
	return "skill_" + strings.ReplaceAll(skill, "-", "_")
}
